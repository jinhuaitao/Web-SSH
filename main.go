package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// 配置
const (
	listenAddr = ":8080"
)

// 全局变量
var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有跨域请求
		},
	}
	templates = template.Must(template.ParseGlob("templates/*.html"))
)

// SSH连接信息
type SSHConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Key      string
}

func main() {
	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 路由
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ssh", sshHandler)

	// 启动服务器
	fmt.Printf("WebSSH服务启动在 http://localhost%s\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

// 首页处理
func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", nil)
}

// SSH WebSocket处理
func sshHandler(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket升级失败:", err)
		return
	}
	defer conn.Close()

	// 从WebSocket接收SSH连接信息
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("读取连接信息失败:", err)
		return
	}

	// 解析连接信息
	var config SSHConfig
	err = json.Unmarshal(msg, &config)
	if err != nil {
		sendErrorMessage(conn, "无效的连接信息")
		return
	}

	// 创建SSH客户端配置
	clientConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	// 添加认证方式
	if config.Password != "" {
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(config.Password))
	}

	if config.Key != "" {
		signer, err := ssh.ParsePrivateKey([]byte(config.Key))
		if err != nil {
			sendErrorMessage(conn, "无效的SSH密钥")
			return
		}
		clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
	}

	// 连接到SSH服务器
	sshConn, err := ssh.Dial("tcp", config.Host+":"+config.Port, clientConfig)
	if err != nil {
		sendErrorMessage(conn, "SSH连接失败: "+err.Error())
		return
	}
	defer sshConn.Close()

	// 创建SSH会话
	session, err := sshConn.NewSession()
	if err != nil {
		sendErrorMessage(conn, "创建SSH会话失败: "+err.Error())
		return
	}
	defer session.Close()

	// 请求伪终端
	termWidth, termHeight := 80, 40
	err = session.RequestPty("xterm", termHeight, termWidth, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	if err != nil {
		sendErrorMessage(conn, "请求伪终端失败: "+err.Error())
		return
	}

	// 获取标准输入输出
	stdin, err := session.StdinPipe()
	if err != nil {
		sendErrorMessage(conn, "获取标准输入失败: "+err.Error())
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		sendErrorMessage(conn, "获取标准输出失败: "+err.Error())
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		sendErrorMessage(conn, "获取标准错误失败: "+err.Error())
		return
	}

	// 启动shell
	err = session.Shell()
	if err != nil {
		sendErrorMessage(conn, "启动Shell失败: "+err.Error())
		return
	}

	// 从SSH服务器读取输出并发送到WebSocket
	go func() {
		multiReader := io.MultiReader(stdout, stderr)
		buf := make([]byte, 1024)
		for {
			n, err := multiReader.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Println("读取SSH输出错误:", err)
				}
				break
			}
			err = conn.WriteMessage(websocket.TextMessage, buf[:n])
			if err != nil {
				log.Println("发送WebSocket消息错误:", err)
				break
			}
		}
	}()

	// 从WebSocket读取输入并发送到SSH
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		// 处理特殊命令
		if len(p) > 1 && p[0] == 1 { // Ctrl+A
			// 处理窗口大小调整
			// 格式: [1][宽度][高度]
			if len(p) >= 5 {
				width := int(p[1])<<8 + int(p[2])
				height := int(p[3])<<8 + int(p[4])
				session.WindowChange(height, width)
			}
		} else {
			// 普通输入
			_, err = stdin.Write(p)
			if err != nil {
				log.Println("写入SSH输入错误:", err)
				break
			}
		}
	}
}

// 发送错误消息到WebSocket
func sendErrorMessage(conn *websocket.Conn, message string) {
	conn.WriteMessage(websocket.TextMessage, []byte("\033[31m"+message+"\033[0m"))
}