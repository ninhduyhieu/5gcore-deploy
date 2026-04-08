IMAGE_NAME = "ubuntu2204"
IMAGE_VER  = "0.0.1"

SERVER_NUM = 1
SERVER_CPU = 4
SERVER_MEM = 4096

AGENT_NUM  = 2
AGENT_CPU  = 4
AGENT_MEM  = 2048

IP_BASE = "192.168.58."

VAGRANT_DISABLE_VBOXSYMLINKCREATE = 1

Vagrant.configure("2") do |config|
  config.ssh.insert_key = false

  (1..SERVER_NUM).each do |i|
    config.vm.define "k3s-server-#{i}" do |server|
      server.vm.box = IMAGE_NAME
      # server.vm.box_version = IMAGE_VER
      server.vm.network "private_network", ip: "#{IP_BASE}#{i + 10}"
      server.vm.hostname = "k3s-server-#{i}"
      server.vm.provider "virtualbox" do |v|
        v.memory = SERVER_MEM
        v.cpus   = SERVER_CPU
      end
    end
  end

  (1..AGENT_NUM).each do |j|
    config.vm.define "k3s-agent-#{j}" do |agent|
      agent.vm.box = IMAGE_NAME
      # agent.vm.box_version = IMAGE_VER
      agent.vm.network "private_network", ip: "#{IP_BASE}#{j + 10 + SERVER_NUM}"
      agent.vm.hostname = "k3s-agent-#{j}"
      agent.vm.provider "virtualbox" do |v|
        v.memory = AGENT_MEM
        v.cpus   = AGENT_CPU
      end
    end
  end
end
