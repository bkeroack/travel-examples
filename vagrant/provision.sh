#!/bin/bash

# Postgres 9.4 isn't available on Ubuntu trusty yet, so we need to add the PPA
/vagrant/vagrant/apt.postgresql.org.sh

apt-get dist-upgrade -y
apt-get install -y httpie curl wget git htop postgresql golang
apt-get autoremove -y

cp /vagrant/vagrant/pg_hba.conf /etc/postgresql/9.4/main/pg_hba.conf

service postgresql restart

sleep 2

sudo -u postgres psql -U postgres -d postgres -c "alter user postgres with password 'postgres';"
sudo -u postgres psql -U postgres -d postgres -c "create database keyvalue with owner postgres;"

echo 'export GOPATH="/home/vagrant/go"' >> /home/vagrant/.bash_profile
chown vagrant /home/vagrant/.bash_profile