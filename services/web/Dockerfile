FROM nginx:1.25-alpine

# Remove default config
RUN rm /etc/nginx/conf.d/default.conf

# Copy custom config
COPY web/nginx.conf /etc/nginx/conf.d/default.conf

# Copy static files
COPY web/index.html /usr/share/nginx/html/index.html

EXPOSE 80
