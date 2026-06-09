IMAGE_NAME = "ubuntu2204"
IMAGE_VER  = "0.0.1"

SERVER_NUM = 1
SERVER_CPU = 4
SERVER_MEM = 4096

AGENT_NUM  = 2
AGENT_CPU  = 4
AGENT_MEM  = 4096

IP_BASE = "192.168.58."

VAGRANT_DISABLE_VBOXSYMLINKCREATE = 1

# Fixed SSH ports keep the self-hosted runner, Ansible and CI/CD stable.
# If a port is already used, Vagrant will fail instead of silently changing it.
SSH_PORTS = {
  "k3s-server-1" => 2200,
  "k3s-agent-1"  => 2201,
  "k3s-agent-2"  => 2202
}

Vagrant.configure("2") do |config|
  config.ssh.insert_key = false
  config.vm.boot_timeout = 900

  (1..SERVER_NUM).each do |i|
    config.vm.define "k3s-server-#{i}" do |server|
      vm_name = "k3s-server-#{i}"
      server.vm.box = IMAGE_NAME
      server.vm.network "private_network", ip: "#{IP_BASE}#{i + 10}"
      server.vm.network "forwarded_port", guest: 22, host: SSH_PORTS[vm_name], id: "ssh", auto_correct: false
      server.vm.hostname = vm_name
      server.vm.provider "virtualbox" do |v|
        v.name   = "5gcore-deploy_#{vm_name}"
        v.memory = SERVER_MEM
        v.cpus   = SERVER_CPU
      end
    end
  end

  (1..AGENT_NUM).each do |j|
    config.vm.define "k3s-agent-#{j}" do |agent|
      vm_name = "k3s-agent-#{j}"
      agent.vm.box = IMAGE_NAME
      agent.vm.network "private_network", ip: "#{IP_BASE}#{j + 10 + SERVER_NUM}"
      agent.vm.network "forwarded_port", guest: 22, host: SSH_PORTS[vm_name], id: "ssh", auto_correct: false
      agent.vm.hostname = vm_name
      agent.vm.provider "virtualbox" do |v|
        v.name   = "5gcore-deploy_#{vm_name}"
        v.memory = AGENT_MEM
        v.cpus   = AGENT_CPU
      end
    end
  end
end
