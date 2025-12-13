echo "127.0.0.1  api.domain-that-makes.no-sense" >> /etc/hosts
echo "127.0.0.1  hub.domain-that-makes.no-sense" >> /etc/hosts
echo "127.0.0.1  ostree.domain-that-makes.no-sense" >> /etc/hosts
TEST_READY=1 go test -v ./server/mdns/... -args count=1
