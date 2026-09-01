ARG BUILDPLATFORM=linux/amd64

FROM --platform=$BUILDPLATFORM python:3.14.7-alpine@sha256:0ad7f98a97b1b8fcc226f5cbe49f0b95cd6f624582cdc6fbf7e41312075cb401 AS base

FROM base AS builder

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

RUN apk update \
    && apk add --no-cache g++ linux-headers \
    && rm -rf /var/cache/apk/*

WORKDIR /build
COPY requirements.txt .

RUN pip install --prefix=/install --no-cache-dir -r requirements.txt

FROM base

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

RUN apk update \
    && apk add --no-cache libstdc++ \
    && rm -rf /var/cache/apk/*

WORKDIR /reorderservice

COPY --from=builder /install /usr/local

COPY src/ ./src/

ENV PYTHONPATH="/reorderservice/src"

ENTRYPOINT ["python", "-m", "reorderservice.main"]
