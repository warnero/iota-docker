FROM golang

RUN apt-get update && apt-get install -y openssh-server supervisor
RUN echo 'root:testpass' | chpasswd
RUN mkdir -p /var/run/sshd /var/log/supervisor
RUN sed -i 's/PermitRootLogin without-password/PermitRootLogin yes/' /etc/ssh/sshd_config

COPY supervisord.go.conf /etc/supervisor/conf.d/supervisord.conf

# copy app source over from system
ADD . /go/src/github.com/asm-products/iota-docker

EXPOSE 22 8080

CMD ["/usr/bin/supervisord", "-n"]