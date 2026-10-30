// Package impl 提供表单留言用例的默认实现：
// 校验入参、读写 xqyun_form_submissions，并映射为接口层 DTO。
// 前台提交实现 submission.Submitter；后台列表 / 详情 / 状态是具体方法。
package impl

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"suki/xqyun/app/core/response"
	"suki/xqyun/app/dto"
	"suki/xqyun/app/models"
	submissionrepository "suki/xqyun/app/repository/submission"
	submission "suki/xqyun/app/services/submission"
)

var validStatuses = map[string]struct{}{
	submission.StatusNew:        {},
	submission.StatusProcessing: {},
	submission.StatusDone:       {},
	submission.StatusSpam:       {},
}

// Service 是表单留言用例的默认实现。
type Service struct {
	submissions *submissionrepository.Repository
}

var _ submission.Submitter = (*Service)(nil)

// New 创建表单留言用例服务。
func New(submissions *submissionrepository.Repository) *Service {
	return &Service{submissions: submissions}
}

// List 分页查询留言，并返回各状态数量。
func (s *Service) List(
	ctx context.Context,
	input submission.ListInput,
) (dto.AdminSubmissionListResponse, error) {
	filter, err := normalizeFilter(input)
	if err != nil {
		return dto.AdminSubmissionListResponse{}, err
	}

	rows, total, err := s.submissions.List(ctx, filter)
	if err != nil {
		return dto.AdminSubmissionListResponse{}, err
	}
	counts, err := s.submissions.StatusCounts(ctx, filter)
	if err != nil {
		return dto.AdminSubmissionListResponse{}, err
	}

	items := make([]dto.AdminSubmission, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAdminSubmission(row))
	}

	return dto.AdminSubmissionListResponse{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Stats: dto.AdminSubmissionStats{
			Total:      counts.Total,
			New:        counts.New,
			Processing: counts.Processing,
			Done:       counts.Done,
			Spam:       counts.Spam,
		},
	}, nil
}

// UnreadCount 返回待处理（status = new）留言数量，供后台侧边栏红点轮询。
// 只有这一个实现、没有替换需求，因此不抽到 services/submission 的契约接口里。
func (s *Service) UnreadCount(ctx context.Context) (int64, error) {
	return s.submissions.CountByStatus(ctx, submission.StatusNew)
}

// Detail 查询留言详情。
func (s *Service) Detail(ctx context.Context, id uint64) (dto.AdminSubmission, error) {
	row, err := s.submissions.FindByID(ctx, id)
	if err != nil {
		return dto.AdminSubmission{}, submissionRepositoryError(err)
	}
	return toAdminSubmission(*row), nil
}

// UpdateStatus 标记留言处理状态。
func (s *Service) UpdateStatus(
	ctx context.Context,
	id uint64,
	status string,
	now int64,
) (dto.AdminSubmission, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if _, ok := validStatuses[normalized]; !ok {
		return dto.AdminSubmission{}, badRequest("留言状态不正确")
	}
	if err := s.submissions.UpdateStatus(ctx, id, normalized, now); err != nil {
		return dto.AdminSubmission{}, submissionRepositoryError(err)
	}
	return s.Detail(ctx, id)
}

// Export 查询符合筛选条件的全部留言，用于导出 CSV。
func (s *Service) Export(
	ctx context.Context,
	input submission.ListInput,
) ([]dto.AdminSubmission, error) {
	filter, err := normalizeFilter(input)
	if err != nil {
		return nil, err
	}
	filter.Page = 1
	filter.PageSize = 10000

	rows, _, err := s.submissions.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdminSubmission, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAdminSubmission(row))
	}
	return items, nil
}

func normalizeFilter(input submission.ListInput) (submissionrepository.Filter, error) {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != "" {
		if _, ok := validStatuses[status]; !ok {
			return submissionrepository.Filter{}, badRequest("留言状态不正确")
		}
	}

	formKey := strings.TrimSpace(input.FormKey)
	if len(formKey) > 50 {
		return submissionrepository.Filter{}, badRequest("表单标识不能超过 50 个字符")
	}

	locale, err := normalizeLocale(input.Locale)
	if err != nil {
		return submissionrepository.Filter{}, err
	}

	if input.CreatedFrom < 0 || input.CreatedTo < 0 {
		return submissionrepository.Filter{}, badRequest("时间筛选参数不正确")
	}
	if input.CreatedFrom > 0 && input.CreatedTo > 0 && input.CreatedFrom > input.CreatedTo {
		return submissionrepository.Filter{}, badRequest("开始时间不能晚于结束时间")
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return submissionrepository.Filter{
		Status:      status,
		FormKey:     formKey,
		Locale:      locale,
		Keyword:     strings.TrimSpace(input.Keyword),
		CreatedFrom: input.CreatedFrom,
		CreatedTo:   input.CreatedTo,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func normalizeLocale(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "":
		return "", nil
	case "zh", "zh-CN":
		return "zh-CN", nil
	case "en":
		return "en", nil
	default:
		return "", badRequest("语言参数不正确")
	}
}

func toAdminSubmission(row models.FormSubmission) dto.AdminSubmission {
	data := map[string]any{}
	if len(row.Data) > 0 {
		_ = json.Unmarshal(row.Data, &data)
	}

	return dto.AdminSubmission{
		ID:        strconv.FormatUint(row.ID, 10),
		FormKey:   row.FormKey,
		Locale:    stringValue(row.Locale),
		Name:      stringValue(row.Name),
		Email:     stringValue(row.Email),
		Phone:     stringValue(row.Phone),
		Company:   stringValue(row.Company),
		Message:   stringValue(row.Message),
		Data:      data,
		Status:    row.Status,
		IP:        stringValue(row.IP),
		UserAgent: stringValue(row.UserAgent),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func submissionRepositoryError(err error) error {
	if errors.Is(err, submissionrepository.ErrNotFound) {
		return submission.ErrNotFound
	}
	return err
}

// badRequest 构造参数错误，返回给控制器后按统一响应输出。
func badRequest(message string) *submission.Error {
	return &submission.Error{Code: response.CodeBadRequest, Message: message}
}
