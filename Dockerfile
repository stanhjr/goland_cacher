# Используем официальный образ Golang
FROM golang:latest

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы go.mod и go.sum и устанавливаем зависимости
COPY go.mod .
COPY go.sum .
RUN go mod download

ARG REDIS_ADDR
ARG REDIS_PASSWORD
# Копируем файлы проекта в текущую директорию
COPY . .

# Собираем Go-приложение
RUN go build -o main .

# Указываем порт, который будет прослушивать приложение
EXPOSE 8080

# Команда для запуска приложения
CMD ["./main"]

