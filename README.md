## Usage ###

#### Запуск 
    $ cd /serverSocks4Proxy
    $ go build
    $ ./serverSocks4Proxy
    
#### Остановка ####

    ctrl+C
    
#### Подготовка ####

    $ sudo ifconfig ens33:0 192.168.234.1
    $ sudo route add -net 192.168.234.0 netmask 255.255.255.0 ens33:0
    $ export https_proxy="socks4://192.168.234.1:8081"
    $ export http_proxy="socks4://192.168.234.1:8081"

#### Тестирование ####

    $ curl https://google.com
