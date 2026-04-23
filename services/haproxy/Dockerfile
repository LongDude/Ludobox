FROM haproxy:2.9-alpine

USER root

COPY auth_validate.lua /usr/local/etc/haproxy/auth_validate.lua
COPY templates /usr/local/etc/haproxy/templates
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
