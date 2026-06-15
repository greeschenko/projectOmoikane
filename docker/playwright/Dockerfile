FROM node:22-alpine

RUN apk add --no-cache \
      libstdc++ \
      nss \
      freetype \
      harfbuzz \
      ca-certificates \
      ttf-freefont \
      libc6-compat

ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=0

RUN npm install -g @playwright/test@1.60.0 && \
    npx playwright install chromium

WORKDIR /e2e
