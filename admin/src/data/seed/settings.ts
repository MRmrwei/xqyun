import type { SiteSettings } from '~/domain/types'

export const settings: SiteSettings = {
  nav: [
    { key: 'nav.products', path: '/products', zh: '产品', en: 'Product', visible: true },
    { key: 'nav.solutions', path: '/solutions', zh: '解决方案', en: 'Solutions', visible: true },
    { key: 'nav.cases', path: '/cases', zh: '客户案例', en: 'Customers', visible: true },
    { key: 'nav.news', path: '/news', zh: '新闻动态', en: 'Journal', visible: true },
    { key: 'nav.about', path: '/about', zh: '关于我们', en: 'About', visible: true }
  ],
  header: {
    showLanguage: true,
    sticky: true,
    showAnnouncement: true,
    announcement: {
      text: { zh: '小鹊云 3.0 已发布', en: 'Magpie Cloud 3.0 is here' },
      highlight: { zh: '让 AI 参与内容工作流', en: 'AI, woven into your content workflow' },
      linkLabel: { zh: '查看新特性', en: 'See what’s new' },
      linkHref: '/news'
    },
    ctas: [
      {
        key: 'header.cta.demo',
        label: { zh: '预约演示', en: 'Book a demo' },
        href: '/contact',
        variant: 'primary',
        visible: true
      },
      {
        key: 'header.cta.cases',
        label: { zh: '客户案例', en: 'Customer stories' },
        href: '/cases',
        variant: 'secondary',
        visible: true
      },
      {
        key: 'header.cta.login',
        label: { zh: '登录控制台', en: 'Sign in' },
        href: '/admin',
        variant: 'ghost',
        visible: true
      }
    ]
  },
  footer: {
    showContact: true,
    cta: {
      label: { zh: '预约演示', en: 'Book a demo' },
      path: '/contact'
    },
    showWordmark: true,
    wordmark: 'MAGPIE',
    columns: [
      {
        key: 'footer.col.products',
        title: { zh: '产品', en: 'Product' },
        visible: true,
        links: [
          { key: 'footer.products.cms', path: '/products', zh: '小鹊云 CMS', en: 'Magpie CMS', visible: true },
          { key: 'footer.products.sites', path: '/products', zh: '小鹊建站', en: 'Magpie Sites', visible: true },
          { key: 'footer.products.insights', path: '/products', zh: '小鹊洞察', en: 'Magpie Insights', visible: true },
          { key: 'footer.products.changelog', path: '/news', zh: '更新日志', en: 'Changelog', visible: true }
        ]
      },
      {
        key: 'footer.col.solutions',
        title: { zh: '解决方案', en: 'Solutions' },
        visible: true,
        links: [
          { key: 'footer.solutions.manufacturing', path: '/solutions', zh: '智能制造', en: 'Manufacturing', visible: true },
          { key: 'footer.solutions.professional', path: '/solutions', zh: '专业服务', en: 'Professional services', visible: true },
          { key: 'footer.solutions.brand', path: '/solutions', zh: '消费品牌', en: 'Consumer brands', visible: true },
          { key: 'footer.solutions.content', path: '/solutions', zh: '内容团队', en: 'Content teams', visible: true }
        ]
      },
      {
        key: 'footer.col.company',
        title: { zh: '公司', en: 'Company' },
        visible: true,
        links: [
          { key: 'footer.company.about', path: '/about', zh: '关于我们', en: 'About', visible: true },
          { key: 'footer.company.cases', path: '/cases', zh: '客户案例', en: 'Customers', visible: true },
          { key: 'footer.company.news', path: '/news', zh: '新闻动态', en: 'Journal', visible: true },
          { key: 'footer.company.join', path: '/contact', zh: '加入我们', en: 'Careers', visible: true }
        ]
      }
    ],
    bottomLinks: [
      { key: 'footer.privacy', path: '/about', zh: '隐私政策', en: 'Privacy', visible: true },
      { key: 'footer.terms', path: '/about', zh: '服务条款', en: 'Terms', visible: true }
    ]
  },
  brand: {
    logo: '',
    accent: '#E8503A',
    background: '#F6F2EA',
    darkBackground: '#141518'
  },
  siteName: { zh: '小鹊云', en: 'Magpie Cloud' },
  companyName: { zh: '小鹊科技', en: 'Magpie Tech' },
  tagline: {
    zh: '让品牌官网成为增长引擎。小鹊云为成长型企业提供区块化的官网搭建与内容云服务。',
    en: 'Turn your website into a growth engine. Magpie Cloud gives growing companies a block-based site builder and content cloud.'
  },
  email: 'hello@xiaoque.com',
  phone: '400-860-2026',
  phoneEn: '+86 400 860 2026',
  address: {
    zh: '上海市徐汇区西岸智塔 18F',
    en: '18F, West Bund AI Tower, Shanghai'
  },
  workHours: {
    zh: '周一至周五 9:00 – 18:00',
    en: 'Mon – Fri, 9:00 – 18:00 CST'
  },
  icp: '沪ICP备2026001234号',
  copyright: {
    zh: '© 2026 小鹊科技',
    en: '© 2026 Magpie Tech'
  },
  socials: [
    { name: '微信', handle: 'xiaoque-tech' },
    { name: '小红书', handle: '小鹊科技' },
    { name: 'LinkedIn', handle: 'magpie-tech' }
  ],
  defaultLocale: 'zh',
  analytics: '',
  headCode: '<!-- 统计代码 / 自定义 head -->',
  allowIndex: true
}
