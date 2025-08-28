#!/bin/bash

# 检测操作系统
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo "无法检测操作系统，请手动安装Docker"
    exit 1
fi

# 安装Docker
case $OS in
    ubuntu|debian)
        echo "检测到 $OS 系统，开始安装Docker..."
        sudo apt-get update
        sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common
        curl -fsSL https://download.docker.com/linux/$OS/gpg | sudo apt-key add -
        sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/$OS $(lsb_release -cs) stable"
        sudo apt-get update
        sudo apt-get install -y docker-ce docker-ce-cli containerd.io
        ;;
    centos|rhel|fedora)
        echo "检测到 $OS 系统，开始安装Docker..."
        sudo yum install -y yum-utils
        sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        sudo yum install -y docker-ce docker-ce-cli containerd.io
        ;;
    *)
        echo "不支持的操作系统: $OS"
        echo "请参考Docker官方文档手动安装: https://docs.docker.com/engine/install/"
        exit 1
        ;;
esac

# 启动Docker服务
sudo systemctl start docker
sudo systemctl enable docker

# 安装Docker Compose
echo "安装Docker Compose..."
COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep 'tag_name' | cut -d\" -f4)
sudo curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 添加当前用户到docker组
sudo usermod -aG docker $USER
echo "Docker安装完成！"
echo "请注销并重新登录，或者重启系统以应用更改"
echo "然后运行 'docker-compose up -d' 启动WebSSH应用"