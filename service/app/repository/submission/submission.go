// Package submission 提供表单留言的数据访问。
package submission

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"suki/xqyun/app/models"
)

// ErrNotFound 表示留言不存在。
var ErrNotFound = errors.New("留言不存在")

// Filter 是留言列表 / 导出查询条件。
type Filter struct {
	Status      string
	FormKey     string
	Locale      string
	Keyword     string
	CreatedFrom int64
	CreatedTo   int64
	Page        int
	PageSize    int
}

// StatusCounts 是各处理状态的留言数量。
type StatusCounts struct {
	Total      int64
	New        int64
	Processing int64
	Done       int64
	Spam       int64
}

// Repository 是表单留言的 GORM 数据访问对象。
type Repository struct {
	db *gorm.DB
}

// New 创建表单留言数据访问对象。
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List 分页查询留言，按提交时间倒序。
func (r *Repository) List(ctx context.Context, filter Filter) ([]models.FormSubmission, int64, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("数据访问层：数据库未初始化")
	}

	var total int64
	countQuery := applyFilters(
		r.db.WithContext(ctx).Table("xqyun_form_submissions AS s"),
		filter,
	)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询留言总数失败：%w", err)
	}

	var rows []models.FormSubmission
	query := applyFilters(
		r.db.WithContext(ctx).Table("xqyun_form_submissions AS s"),
		filter,
	)
	err := query.
		Select("s.*").
		Order("s.created_at DESC, s.id DESC").
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询留言列表失败：%w", err)
	}
	return rows, total, nil
}

// FindByID 按主键查询留言详情。
func (r *Repository) FindByID(ctx context.Context, id uint64) (*models.FormSubmission, error) {
	if r.db == nil {
		return nil, fmt.Errorf("数据访问层：数据库未初始化")
	}

	var row models.FormSubmission
	err := r.db.WithContext(ctx).
		Table("xqyun_form_submissions AS s").
		Select("s.*").
		Where("s.id = ?", id).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询留言失败：%w", err)
	}
	return &row, nil
}

// StatusCounts 统计各状态留言数量。状态筛选不参与统计，其余筛选条件生效。
func (r *Repository) StatusCounts(ctx context.Context, filter Filter) (StatusCounts, error) {
	if r.db == nil {
		return StatusCounts{}, fmt.Errorf("数据访问层：数据库未初始化")
	}

	var rows []struct {
		Status string
		Count  int64
	}
	statusFilter := filter
	statusFilter.Status = ""
	query := applyFilters(
		r.db.WithContext(ctx).Table("xqyun_form_submissions AS s"),
		statusFilter,
	)
	err := query.
		Select("s.status AS status, COUNT(*) AS count").
		Group("s.status").
		Scan(&rows).Error
	if err != nil {
		return StatusCounts{}, fmt.Errorf("统计留言状态失败：%w", err)
	}

	var counts StatusCounts
	for _, row := range rows {
		counts.Total += row.Count
		switch row.Status {
		case "new":
			counts.New = row.Count
		case "processing":
			counts.Processing = row.Count
		case "done":
			counts.Done = row.Count
		case "spam":
			counts.Spam = row.Count
		}
	}
	return counts, nil
}

// CountByStatus 统计指定处理状态的留言数量。
// 供后台侧边栏红点轮询使用，只返回一个数字，不做列表与聚合。
func (r *Repository) CountByStatus(ctx context.Context, status string) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("数据访问层：数据库未初始化")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.FormSubmission{}).
		Where("status = ?", status).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("统计留言数量失败：%w", err)
	}
	return count, nil
}

// Create 保存一条新的表单留言。
func (r *Repository) Create(ctx context.Context, row *models.FormSubmission) error {
	if r.db == nil {
		return fmt.Errorf("数据访问层：数据库未初始化")
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("保存留言失败：%w", err)
	}
	return nil
}

// UpdateStatus 更新留言处理状态与处理时间。
func (r *Repository) UpdateStatus(ctx context.Context, id uint64, status string, now int64) error {
	if r.db == nil {
		return fmt.Errorf("数据访问层：数据库未初始化")
	}

	result := r.db.WithContext(ctx).
		Model(&models.FormSubmission{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"updated_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("更新留言状态失败：%w", result.Error)
	}
	if result.RowsAffected == 0 {
		// 状态与原文相同时 MySQL 可能返回 0，需要再确认留言是否存在。
		var count int64
		if err := r.db.WithContext(ctx).
			Model(&models.FormSubmission{}).
			Where("id = ?", id).
			Count(&count).Error; err != nil {
			return fmt.Errorf("确认留言是否存在失败：%w", err)
		}
		if count == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func applyFilters(query *gorm.DB, filter Filter) *gorm.DB {
	if filter.Status != "" {
		query = query.Where("s.status = ?", filter.Status)
	}
	if filter.FormKey != "" {
		query = query.Where("s.form_key = ?", filter.FormKey)
	}
	if filter.Locale != "" {
		query = query.Where("s.locale = ?", filter.Locale)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"(s.name LIKE ? OR s.email LIKE ? OR s.phone LIKE ? OR s.company LIKE ? OR s.message LIKE ?)",
			like,
			like,
			like,
			like,
			like,
		)
	}
	if filter.CreatedFrom > 0 {
		query = query.Where("s.created_at >= ?", filter.CreatedFrom)
	}
	if filter.CreatedTo > 0 {
		query = query.Where("s.created_at <= ?", filter.CreatedTo)
	}
	return query
}
