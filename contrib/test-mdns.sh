apt update && apt install -y libnss-mdns iproute2

ip link add dummy224 type dummy
trap "ip link del dummy224" EXIT
ip link set dummy224 multicast on up
ip addr add 224.0.0.251/24 dev dummy224
ip route add 224.0.0.251/32 dev dummy224

sed -i s/mdns4_minimal/mdns4/g /etc/nsswitch.conf
echo ".domain-that-makes.no-sense." >> /etc/mdns.allow
echo ".domain-that-makes.no-sense" >> /etc/mdns.allow

TEST_READY=1 go test -tags=mdns -v ./server/mdns/... -args count=1
