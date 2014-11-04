#!/bin/bash

cleanup() {
    apt-get -y autoremove
    apt-get -y autoclean
    apt-get -y clean
}

pkg_update(){
    export DEBIAN_FRONTEND=noninteractive

    # update
    apt-get -y update
    apt-get -y --force-yes upgrade
    
    cleanup
}

install_docker() {
    # perform some very rudimentary platform detection
    local lsb_dist=''
    if [[ -z "$lsb_dist" ]] && [[ -r /etc/lsb-release ]]; then
        local lsb_dist="$(. /etc/lsb-release && echo "$DISTRIB_ID")"
    fi
    if [[ -z "$lsb_dist" ]] && [ -r /etc/debian_version ]; then
        local lsb_dist='Debian'
    fi
    
    local lsb_dist="$(echo "$lsb_dist" | tr '[:upper:]' '[:lower:]')"
    case "$lsb_dist" in
        debian)
            # get systemd files
            wget http://jesss.s3.amazonaws.com/docker/systemd/docker.service
            wget http://jesss.s3.amazonaws.com/docker/systemd/docker.socket

            mv docker.service /lib/systemd/system/
            mv docker.socket /lib/systemd/system/

            # restart systemd daemon
            systemctl daemon-reload

            # enable docker on boot
            systemctl enable docker
            ;;
        ubuntu)
            # get upstart script
            wget http://jesss.s3.amazonaws.com/docker/upstart/docker.conf

            mv docker.conf /etc/init/
            ;;
        *)
            echo "wtf is this if it's not debian or ubuntu"
            ;;
    esac

    # get bash completions
    mkdir -p /etc/bash_completion.d
    wget https://raw.githubusercontent.com/docker/docker/master/contrib/completion/bash/docker
    cp docker /etc/bash_completion.d/

    # get latest binary
    wget https://get.docker.com/builds/Linux/x86_64/docker-latest -O docker
    chmod +x docker
    mv docker /usr/bin/docker

    # add the docker group
    groupadd docker
    gpasswd -a ${USER} docker

    # start the docker service
    if [[ $lsb_dist == "debian" ]]; then
        apt-get install -y --force-yes cgroupfs-mount
        systemctl start docker
    elif [[ $lsb_dist == "ubuntu" ]]; then
        apt-get install -y --force-yes cgroup-lite
        service docker start
    fi

    # install aufs headers
    apt-get install -y --force-yes linux-image-extra-`uname -r`

    echo "Docker has been installed. If you want memory management & swap"
    echo "add this to /etc/default/grub, run update-grub & reboot: "
    echo 'GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1"'
}

install_base() {
    pkg_update

    local pkgs="adduser
apparmor
aufs-tools
ca-certificates
cron
curl
htop
iptables
less
libapparmor1
libsqlite3-0
locales
mount
ssh
sudo
tar
tree
tzdata
unzip
wget
vim-nox
"

    printf %s "$pkgs" | while IFS= read -r pkg; do
        echo "Installing $pkg"
        DEBIAN_FRONTEND=noninteractive apt-get install -y --force-yes $pkg --no-install-recommends
    done

    cleanup

    install_docker
}

setup_nginx() {
    # get the nginx configs
    mkdir -p /root/nginx/conf.d
    cd /root/nginx

    if [[ ! -f nginx.conf ]]; then
        wget https://raw.githubusercontent.com/docker/docker/meta/servers/nginx/nginx.conf
    fi
    if [[ ! -f nginx.conf ]]; then
        wget https://raw.githubusercontent.com/docker/docker/meta/servers/nginx/mime.types
    fi

    cd conf.d
    if [[ ! -f ci.dockerproject.com.conf ]]; then
        wget https://raw.githubusercontent.com/docker/docker/meta/servers/nginx/conf.d/ci.dockerproject.com.conf
    fi

    cd
    if [[ -d /root/nginx/ssl ]]; then
        # run the container
        docker run -d --name nginx --restart always -p 80:80 -p 443:443 -v /root/nginx:/etc/nginx nginx
    else
        echo "You need the ssl certs :), get the nginx/ssl directory and run this function again."
    fi
}

usage() {
    echo "Usage:"
    echo "  cmd pkg-update      - update apt-packages for intital server"
    echo "  cmd install-base    - install base packages (curl, docker, vim, etc)"
    echo "  cmd install-docker  - install docker"
    echo "  cmd setup-nginx     - setup nginx container for drone"
    exit 1
}

main() {
    # check to make sure they passed the right
    # amount of arguments
    if [[ $# -lt 1 ]]; then
        usage;
    fi

    case "$1" in
        upgrade-kernel) 
            upgrade_kernel
            ;;
        install-base)
            install_base
            ;;
        install-docker)
            install_docker
            ;;
        setup-nginx)
            setup_nginx
            ;;
        *)
            usage
            ;;
    esac
}

main $@
