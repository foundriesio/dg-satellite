FROM public.ecr.aws/docker/library/alpine:3.22

RUN apk add avahi && rm /etc/avahi/services/*
COPY avahi-daemon.conf /etc/avahi/
COPY device-gateway.service /etc/avahi/services/

CMD avahi-daemon
