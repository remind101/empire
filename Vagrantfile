# -*- mode: ruby -*-
# # vi: set ft=ruby :

# Adds a user-data file to the machine.
def cloud_config(config)
  config.vm.provision :file, source: File.expand_path('../cluster/user-data', __FILE__), destination: '/tmp/vagrantfile-user-data'
  config.vm.provision :shell, inline: 'mv /tmp/vagrantfile-user-data /var/lib/coreos-vagrant/', privileged: true
end

# Adds docker registry authentication.
def docker_auth(config)
  config.vm.provision :file, source: File.expand_path('~/.dockercfg'), destination: '/tmp/dockercfg'
  config.vm.provision :shell, inline: 'mv /tmp/dockercfg /home/core/.dockercfg', privileged: true
end

Vagrant.configure('2') do |config|
  # always use Vagrants insecure key
  config.ssh.insert_key = false

  config.vm.box = 'coreos-stable'
  config.vm.box_version = '>= 308.0.1'
  config.vm.box_url = 'http://stable.release.core-os.net/amd64-usr/current/coreos_production_vagrant.json'

  config.vm.provider :virtualbox do |v|
    # On VirtualBox, we don't have guest additions or a functional vboxsf
    # in CoreOS, so tell Vagrant that so it can be smarter.
    v.check_guest_additions = false
    v.functional_vboxsf     = false
  end

  # plugin conflict
  config.vbguest.auto_update = false if Vagrant.has_plugin?('vagrant-vbguest')

  config.vm.synced_folder '.', '/home/core/share', id: 'core', nfs: true, mount_options: ['nolock,vers=3,udp']

  config.vm.define 'c1' do |c1|
    c1.vm.hostname = 'c1'
    c1.vm.network :private_network, ip: '172.20.20.10'

    cloud_config c1
    docker_auth c1
  end

  #config.vm.define 'm1' do |m1|
    #m1.vm.hostname = 'm1'
    #m1.vm.network :private_network, ip: '172.20.20.11'

    #cloud_config m1
    #docker_auth m1
  #end
end
