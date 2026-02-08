package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sunnyproxy/internal/logger"
	"sunnyproxy/internal/proxy"
	"sunnyproxy/internal/rules"
	"sunnyproxy/internal/web"
	"sunnyproxy/pkg/config"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("加载配置文件失败，使用默认配置: %v\n", err)
		cfg = config.Default()
	}

	// 支持 Render/Railway 等平台的 PORT 环境变量
	// Railway 模式：单端口同时提供 Web 和代理服务
	singlePortMode := false
	if port := os.Getenv("PORT"); port != "" {
		var p int
		fmt.Sscanf(port, "%d", &p)
		if p > 0 {
			cfg.Server.WebPort = p
			cfg.Server.ProxyPort = p // 单端口模式
			singlePortMode = true
		}
	}

	broadcaster := logger.GetBroadcaster()
	broadcaster.SetConsoleOutput(cfg.Logging.Console)

	engine, err := rules.NewEngine(cfg.Rules.File)
	if err != nil {
		log.Printf("初始化规则引擎失败: %v\n", err)
		engine, _ = rules.NewEngine("rules.json")
	}

	wrapper := proxy.NewWrapper()
	wrapper.SetPort(cfg.Server.ProxyPort)

	caCert, caKey, err := loadOrGenerateCA()
	if err != nil {
		log.Printf("生成 CA 证书失败: %v\n", err)
	} else {
		if err := wrapper.SetCA(caCert, caKey); err != nil {
			log.Printf("设置 CA 证书失败: %v\n", err)
		}
	}

	wrapper.EnableMITM()

	handler := proxy.NewHandler(engine)
	handler.SetupHandlers(wrapper.GetProxy())

	webServer := web.NewServer(cfg, engine, wrapper)

	if singlePortMode {
		// 单端口模式：组合 Web 和代理
		combinedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 代理请求特征：有完整的 URL 或者是 CONNECT 方法
			if r.Method == "CONNECT" || r.URL.Host != "" {
				wrapper.GetProxy().ServeHTTP(w, r)
			} else {
				webServer.GetHandler().ServeHTTP(w, r)
			}
		})

		go func() {
			addr := fmt.Sprintf("%s:%d", cfg.Server.BindIP, cfg.Server.WebPort)
			log.Printf("启动单端口服务（Web+代理），地址: %s\n", addr)
			if err := http.ListenAndServe(addr, combinedHandler); err != nil {
				log.Fatalf("服务启动失败: %v\n", err)
			}
		}()
	} else {
		// 双端口模式
		go func() {
			if err := webServer.Start(); err != nil {
				log.Fatalf("Web 服务启动失败: %v\n", err)
			}
		}()

		go func() {
			addr := fmt.Sprintf("%s:%d", cfg.Server.BindIP, cfg.Server.ProxyPort)
			log.Printf("启动代理服务，地址: %s\n", addr)
			if err := http.ListenAndServe(addr, wrapper.GetProxy()); err != nil {
				log.Fatalf("代理服务启动失败: %v\n", err)
			}
		}()
	}

	log.Println("========================================")
	log.Printf("SunnyProxy 已启动")
	log.Printf("代理端口: %d", cfg.Server.ProxyPort)
	log.Printf("Web 管理: http://%s:%d", cfg.Server.BindIP, cfg.Server.WebPort)
	log.Printf("证书下载: http://%s:%d/ssl", cfg.Server.BindIP, cfg.Server.WebPort)
	log.Println("========================================")

	// 启动定时清理任务
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			// 清理超过 24 小时的 Token
			removed := engine.CleanupOldTokens(24 * time.Hour)
			if removed > 0 {
				log.Printf("[Cleanup] 清理了 %d 个过期 Token", removed)
			}
			// 强制 GC
			// runtime.GC()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭服务...")
	wrapper.Stop()
	log.Println("服务已关闭")
}

func loadOrGenerateCA() ([]byte, []byte, error) {
	certFile := "ca.crt"
	keyFile := "ca.key"

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			cert, err := os.ReadFile(certFile)
			if err != nil {
				return nil, nil, err
			}
			key, err := os.ReadFile(keyFile)
			if err != nil {
				return nil, nil, err
			}
			return cert, key, nil
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"SunnyProxy"},
			OrganizationalUnit: []string{"SunnyProxy CA"},
			CommonName:         "SunnyProxy Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	os.WriteFile(certFile, certPEM, 0644)
	os.WriteFile(keyFile, keyPEM, 0600)

	return certPEM, keyPEM, nil
}
