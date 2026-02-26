package ipfilter

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

type Filter struct {
	enabled       bool
	whitelist     map[string]bool // 白名单IP（已验证的中国IP）
	blacklist     map[string]bool // 黑名单IP（已验证的境外IP）
	blockedCount  map[string]int  // 拦截次数统计
	mu            sync.RWMutex
	whitelistFile string
	geoDB         *geoip2.Reader
}

var (
	filterInstance *Filter
	filterOnce     sync.Once
)

// GeoIP 数据库路径搜索顺序
var geoDBPaths = []string{
	"GeoLite2-Country.mmdb",
	"/opt/sunnyproxy/GeoLite2-Country.mmdb",
	"/usr/share/GeoIP/GeoLite2-Country.mmdb",
	"/var/lib/GeoIP/GeoLite2-Country.mmdb",
}

// 私有/内网IP段（始终允许）
var privateRanges = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

var privateNets []*net.IPNet

func init() {
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			privateNets = append(privateNets, ipNet)
		}
	}
}

func GetFilter() *Filter {
	filterOnce.Do(func() {
		filterInstance = &Filter{
			enabled:       true,
			whitelist:     make(map[string]bool),
			blacklist:     make(map[string]bool),
			blockedCount:  make(map[string]int),
			whitelistFile: "whitelist.txt",
		}

		// 加载 GeoIP 数据库
		filterInstance.loadGeoDB()
		filterInstance.loadWhitelist()
		log.Printf("[IPFilter] IP过滤器已启动，仅允许中国IP访问")
		log.Printf("[IPFilter] 白名单已加载 %d 个IP", len(filterInstance.whitelist))
	})
	return filterInstance
}

// loadGeoDB 加载 GeoIP 数据库
func (f *Filter) loadGeoDB() {
	for _, path := range geoDBPaths {
		db, err := geoip2.Open(path)
		if err == nil {
			f.geoDB = db
			log.Printf("[IPFilter] GeoIP 数据库已加载: %s", path)
			return
		}
	}
	log.Printf("[IPFilter] 警告: 未找到 GeoIP 数据库，将仅使用白名单/黑名单过滤")
}

// loadWhitelist 从文件加载白名单
func (f *Filter) loadWhitelist() {
	file, err := os.Open(f.whitelistFile)
	if err != nil {
		// 文件不存在，忽略
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip != "" && !strings.HasPrefix(ip, "#") {
			f.whitelist[ip] = true
		}
	}
}

// saveWhitelist 保存白名单到文件
func (f *Filter) saveWhitelist() {
	f.mu.RLock()
	defer f.mu.RUnlock()

	file, err := os.Create(f.whitelistFile)
	if err != nil {
		log.Printf("[IPFilter] 保存白名单失败: %v", err)
		return
	}
	defer file.Close()

	file.WriteString("# SunnyProxy IP Whitelist (已验证的中国IP)\n")
	file.WriteString("# 自动生成，请勿手动编辑\n\n")
	for ip := range f.whitelist {
		file.WriteString(ip + "\n")
	}
}

func (f *Filter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

func (f *Filter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

// isPrivateIP 检查是否是内网IP
func isPrivateIP(ip net.IP) bool {
	for _, ipNet := range privateNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// queryIPLocation 使用本地 GeoIP 数据库查询IP归属地
func (f *Filter) queryIPLocation(ipStr string) (bool, error) {
	if f.geoDB == nil {
		return false, fmt.Errorf("GeoIP 数据库未加载")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("无效IP: %s", ipStr)
	}

	record, err := f.geoDB.Country(ip)
	if err != nil {
		return false, err
	}

	return record.Country.IsoCode == "CN", nil
}

// IsAllowed 检查IP是否允许访问
func (f *Filter) IsAllowed(ipStr string) bool {
	if !f.IsEnabled() {
		return true
	}

	// 解析IP地址（可能带端口）
	host, _, err := net.SplitHostPort(ipStr)
	if err != nil {
		host = ipStr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// 内网IP始终允许
	if isPrivateIP(ip) {
		return true
	}

	// 检查白名单（已验证的中国IP）
	f.mu.RLock()
	if f.whitelist[host] {
		f.mu.RUnlock()
		return true
	}
	// 检查黑名单（已验证的境外IP）
	if f.blacklist[host] {
		f.blockedCount[host]++
		f.mu.RUnlock()
		return false
	}
	f.mu.RUnlock()

	// 新IP，使用 GeoIP 查询归属地
	isChina, err := f.queryIPLocation(host)
	if err != nil {
		// GeoIP 查询失败，默认允许（避免误杀）
		log.Printf("[IPFilter] 查询IP %s 失败: %v，默认允许", host, err)
		return true
	}

	f.mu.Lock()
	if isChina {
		// 中国IP，加入白名单
		f.whitelist[host] = true
		f.mu.Unlock()
		// 异步保存白名单
		go f.saveWhitelist()
		log.Printf("[IPFilter] 新增白名单IP: %s (中国)", host)
		return true
	} else {
		// 境外IP，加入黑名单
		f.blacklist[host] = true
		f.blockedCount[host] = 1
		f.mu.Unlock()
		log.Printf("[IPFilter] 拦截境外IP: %s", host)
		return false
	}
}

// Close 关闭 GeoIP 数据库
func (f *Filter) Close() {
	if f.geoDB != nil {
		f.geoDB.Close()
	}
}

// AddToWhitelist 手动添加IP到白名单
func (f *Filter) AddToWhitelist(ip string) {
	f.mu.Lock()
	f.whitelist[ip] = true
	delete(f.blacklist, ip)
	f.mu.Unlock()
	f.saveWhitelist()
	log.Printf("[IPFilter] 手动添加白名单IP: %s", ip)
}

// RemoveFromWhitelist 从白名单移除IP
func (f *Filter) RemoveFromWhitelist(ip string) {
	f.mu.Lock()
	delete(f.whitelist, ip)
	f.mu.Unlock()
	f.saveWhitelist()
}

// GetWhitelist 获取白名单列表
func (f *Filter) GetWhitelist() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	list := make([]string, 0, len(f.whitelist))
	for ip := range f.whitelist {
		list = append(list, ip)
	}
	return list
}

// GetBlockedIPs 获取被拦截的IP及次数
func (f *Filter) GetBlockedIPs() map[string]int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]int)
	for ip, count := range f.blockedCount {
		result[ip] = count
	}
	return result
}

// GetStats 获取统计信息
func (f *Filter) GetStats() (whitelistCount, blacklistCount, totalBlocked int) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	whitelistCount = len(f.whitelist)
	blacklistCount = len(f.blacklist)
	for _, count := range f.blockedCount {
		totalBlocked += count
	}
	return
}
