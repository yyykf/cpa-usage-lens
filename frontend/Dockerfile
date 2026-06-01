# ---- 构建阶段（固定在 builder 原生架构跑，前端产物是架构无关的静态文件）----
FROM --platform=$BUILDPLATFORM node:20-alpine AS build
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

# ---- 运行阶段（nginx 静态托管 + 反代 /api 到 backend）----
FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
