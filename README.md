# go-musthave-diploma-tpl

Шаблон репозитория для индивидуального дипломного проекта курса «Go-разработчик»

# Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без
   префикса `https://`) для создания модуля

# Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m master template https://github.com/yandex-praktikum/go-musthave-diploma-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/master .github
```

Затем добавьте полученные изменения в свой репозиторий.

# Работа с приложением

## Запуск приложения

Для работы приложения необходимо запустить два сервиса:

1. Основное приложение (Gophermart)
2. Система начислений (Accrual)

### Запуск системы начислений (Accrual)

```bash
./cmd/accrual/accrual_linux_amd64 -a :8081 -d "postgres://host:host@localhost:5432/gophermart?sslmode=disable"
```

Параметры:
- `-a` - адрес и порт для запуска сервера (по умолчанию :8081)
- `-d` - строка подключения к базе данных PostgreSQL

### Запуск основного приложения (Gophermart)

```bash
go run cmd/gophermart/main.go -a :8080 -d "postgres://host:host@localhost:5432/gophermart?sslmode=disable" -r "http://localhost:8081"
```

Параметры:
- `-a` - адрес и порт для запуска сервера (по умолчанию :8080)
- `-d` - строка подключения к базе данных PostgreSQL
- `-r` - адрес системы начислений

## API Endpoints

### Регистрация пользователя
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"login":"username","password":"password"}' \
  http://localhost:8080/api/user/register
```

### Авторизация пользователя
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"login":"username","password":"password"}' \
  http://localhost:8080/api/user/login
```
В ответе в заголовке `Authorization` будет получен JWT токен для последующих запросов.

### Загрузка номера заказа
```bash
curl -X POST -H "Content-Type: text/plain" \
  -H "Authorization: Bearer <token>" \
  -d "12345678903" \
  http://localhost:8080/api/user/orders
```
Номер заказа должен проходить проверку алгоритмом Луна.

### Получение списка заказов пользователя
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/user/orders
```

### Получение текущего баланса
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/user/balance
```

### Списание баллов
```bash
curl -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"order":"12345678903","sum":100}' \
  http://localhost:8080/api/user/balance/withdraw
```

### Получение информации о списаниях
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/user/withdrawals
```

### Получение информации о конкретном заказе
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/orders/{order_num}
```