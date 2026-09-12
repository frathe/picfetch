FROM debian:trixie-slim@sha256:d7e12182ce18b85b93007c1dedf31f2d29e01ccf3182cc4017c709b6259bc132
RUN apt-get update && apt-get install -y --no-install-recommends gource ffmpeg xvfb xauth fonts-dejavu-core python3 python3-pil python3-numpy ca-certificates && rm -rf /var/lib/apt/lists/*
ENV LIBGL_ALWAYS_SOFTWARE=1
ENV TZ=Europe/Brussels
ENV XDG_CACHE_HOME=/tmp/cache
ENV PYTHONDONTWRITEBYTECODE=1
WORKDIR /work
