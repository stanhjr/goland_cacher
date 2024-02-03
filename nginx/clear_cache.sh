find /var/cache/nginx_cache -type f -mmin +1 -exec sh -c 'rm -f "{}" && echo "Removed: {} at $(date)" >> /var/cache/collector.log' \;




