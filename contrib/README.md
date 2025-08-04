# About

This directory contains tools useful for local development

## dev-shell

This script builds a container with all the required dependencies for
developing on this code base and will drop you in a container with the
project source code mounted.

## docker-compose.yml

This compose project will launch a satellite server that devices can
communicate with. In order to use this you must first:
```
 $ go run github.com/foundriesio/dg-satellite/cmd create-csr \
     --datadir .compose-server-data \
     --dnsname <HOSTNAME> --factory <FACTORY>
 $ go run github.com/foundriesio/dg-satellite/cmd \
    --datadir .compose-server-data sign-csr \
    --cakey <PATH TO FACTORY PKI>/factory_ca.key \
    --cacert <PATH TO FACTORY PKI>/factory_ca.pem \
    --csr .compose-server-data/certs/tls.csr
 $ fioctl keys ca show --just-device-cas > .compose-server-data/certs/cas.pem
```
