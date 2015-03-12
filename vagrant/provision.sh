#!/bin/bash

GOVERSION="1.4.2"

# Postgres 9.4 isn't available on Ubuntu trusty yet, so we need to add the PPA
/vagrant/vagrant/apt.postgresql.org.sh

apt-get dist-upgrade -y
apt-get install -y httpie curl wget git htop postgresql
apt-get autoremove -y

cp /vagrant/vagrant/pg_hba.conf /etc/postgresql/9.4/main/pg_hba.conf

service postgresql restart

sleep 2

sudo -u postgres psql -U postgres -d postgres -c "alter user postgres with password 'postgres';"
sudo -u postgres psql -U postgres -d postgres -c "create database keyvalue with owner postgres; create database keyvalue_testing with owner postgres;"

FNAME="go${GOVERSION}.linux-amd64.tar.gz"
wget "https://storage.googleapis.com/golang/${FNAME}" -O /home/vagrant/${FNAME} -nv
tar -C /usr/local -xzf "./${FNAME}"

echo 'export PATH="${PATH}:/usr/local/go/bin"' > /home/vagrant/.bash_profile
echo 'export GOPATH="/home/vagrant/go"' >> /home/vagrant/.bash_profile
echo 'export PS1="${debian_chroot:+($debian_chroot)}\u@\h:\W\$"' >> /home/vagrant/.bash_profile
echo 'cd /home/vagrant/go' >> /home/vagrant/.bash_profile 
chown vagrant /home/vagrant/.bash_profile