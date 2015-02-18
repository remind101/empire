# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

num_minions = 3
minion_base_ip = "192.168.55.40"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.ssh.forward_agent = true

  # Setup controller box
  config.vm.define "controller" do |controller|
    controller.vm.hostname = "controller"
    controller.vm.box = "empire_controller"
    controller.vm.box_url = ["file://#{ENV['HOME']}/Documents/remind_sync/empire_controller.box", "http://empire-image-artifacts.s3-website-us-east-1.amazonaws.com/empire_controller.box"]
    controller.vm.network "private_network", ip: "192.168.55.11"
    controller.vm.provider "virtualbox" do |vb|
        vb.customize [
            "modifyvm", :id,
            "--name", "controller",
            "--memory", "1024",
            "--cpus", "1",
            "--natdnspassdomain1", "on",
            "--natdnsproxy1", "off",
            "--natdnshostresolver1", "on",
        ]
    end
  end

  # Setup minion boxes
  num_minions.times do |i|
    hostname = "minion%d" % [(i+1)]

    # only autostart the first minion
    opts = { autostart: i == 0 }
    config.vm.define :"#{hostname}", opts do |box|
      box.vm.hostname = hostname
      box.vm.box = "empire_minion"
      box.vm.box_url = ["file://#{ENV['HOME']}/Documents/remind_sync/empire_minion.box", "http://empire-image-artifacts.s3-website-us-east-1.amazonaws.com/empire_minion.box"]
      ip = minion_base_ip.split('.').tap{|ip| ip[-1] = (ip[-1].to_i + i).to_s}.join('.')
      box.vm.network "private_network", ip: ip
      box.vm.provider "virtualbox" do |vb|
          vb.customize [
              "modifyvm", :id,
              "--name", hostname,
              "--memory", "1024",
              "--cpus", "1",
              "--natdnspassdomain1", "on",
              "--natdnsproxy1", "off",
              "--natdnshostresolver1", "on",
          ]
      end
    end
  end
end
