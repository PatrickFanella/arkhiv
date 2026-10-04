# Extract the already qualified interface without rebuilding its dependencies.
# SOURCE_IMAGE must identify the consumer's immutable application image.
ARG SOURCE_IMAGE
FROM ${SOURCE_IMAGE} AS consumer
FROM nginx:1.28-alpine
ARG CORE_UPSTREAM
RUN rm /usr/share/nginx/html/50x.html
COPY --from=consumer /app/web /usr/share/nginx/html
COPY deploy/reference-nginx.conf /etc/nginx/nginx.conf
RUN case "$CORE_UPSTREAM" in bill-core:8080|democracy-core:8080) ;; *) exit 1 ;; esac \
    && sed -i "s/archive-core:8080/$CORE_UPSTREAM/" /etc/nginx/nginx.conf
USER 101:101
EXPOSE 8080
HEALTHCHECK --interval=20s --timeout=5s --start-period=10s --retries=3 CMD ["wget", "-qO-", "http://127.0.0.1:8080/healthz"]
ENTRYPOINT ["nginx"]
CMD ["-g", "daemon off;"]
