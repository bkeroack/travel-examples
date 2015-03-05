# -*- mode: ruby -*-
# vi: set ft=ruby :
Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/trusty64"

  config.vm.network "private_network", ip: "192.168.10.10"
  config.vm.synced_folder ENV["GOPATH"], "/home/vagrant/go"

  config.vm.provider "virtualbox" do |vb|
    vb.gui = true
    vb.cpus = 2
    vb.memory = "1024"
  end

  config.vm.provision "shell", path: "vagrant/provision.sh"
end
