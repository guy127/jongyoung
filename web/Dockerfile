# build 3 ขั้น: ติดตั้ง dependency → build → image สำหรับรันที่มีแค่ output standalone
FROM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:24-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_* ถูกฝังลง JavaScript ฝั่ง browser ตอน build จึงต้องส่งเป็น build arg (ไม่ใช่ env ตอนรัน)
ARG NEXT_PUBLIC_API_BASE_URL=http://api.jongyoung.localhost/api/v1
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
USER node
EXPOSE 3000
CMD ["node", "server.js"]
