package domainfilter

import (
	"log"
	"strings"
	"sync"
)

type DomainFilter struct {
	enabled    bool
	allowList  []string // 允许的域名/IP关键词
	mu         sync.RWMutex
}

var (
	filterInstance *DomainFilter
	filterOnce     sync.Once
)

func GetFilter() *DomainFilter {
	filterOnce.Do(func() {
		filterInstance = &DomainFilter{
			enabled: true,
			allowList: []string{
				// IP地址
				"114.66.51.98",
				"110.42.67.153",

				// suraimu
				"suraimu.com",

				// qmai相关
				"qmai",

				// 微信相关
				"weixin.qq.com",
				"wx.qq.com",
				"wechat.com",
				"weixin",
				"weixinbridge",
				"servicewechat",
				"wechatpay.cn",    // 微信支付
				"qlogo.cn",        // 微信头像
				"qpic.cn",         // 微信图片
				"wxpay.cn",        // 微信支付
				"weixinmp.com",    // 微信公众平台
				"wechatapp.com",   // 微信小程序

				// QQ相关（微信依赖）
				"qq.com",

				// 微信支付
				"tenpay.com",
				"mch.weixin",

				// 苹果推送（iOS需要）
				"apple.com",
				"icloud.com",
				"mzstatic.com",

				// 闲鱼相关
				"idle.taobao.com",
				"2.taobao.com",
				"idlefish",
				"xianyu",

				// 阿里相关
				"alibaba.com",
				"alibabacloud.com",
				"aliyun.com",
				"aliyuncs.com",
				"alicdn.com",
				"taobao.com",
				"tmall.com",
				"alipay.com",
				"alipayobjects.com",
				"mmstat.com",
				"tbcdn.cn",
				"aliapp.org",
				"amap.com",
				"uc.cn",
				"ucweb.com",
			},
		}
		log.Printf("[DomainFilter] 域名白名单已启用，允许 %d 个规则", len(filterInstance.allowList))
	})
	return filterInstance
}

// IsAllowed 检查域名是否允许访问
func (f *DomainFilter) IsAllowed(host string) bool {
	if !f.enabled {
		return true
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// 移除端口号
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		// 检查是否是IPv6
		if !strings.Contains(host, "[") {
			host = host[:idx]
		}
	}

	host = strings.ToLower(host)

	// 检查是否匹配白名单
	for _, allowed := range f.allowList {
		if strings.Contains(host, strings.ToLower(allowed)) {
			return true
		}
	}

	return false
}

// SetEnabled 启用/禁用过滤
func (f *DomainFilter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

// IsEnabled 检查是否启用
func (f *DomainFilter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

// AddAllowed 添加允许的域名
func (f *DomainFilter) AddAllowed(domain string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.allowList = append(f.allowList, domain)
}

// GetAllowList 获取白名单
func (f *DomainFilter) GetAllowList() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make([]string, len(f.allowList))
	copy(result, f.allowList)
	return result
}
