# URL Shortener App

Приложение на go, позволяющее сокращать длинные URL-адреса и перенаправлять пользователей по коротким ключам. Оно поддерживает HTTP и gRPC интерфейсы, а также позволяет использовать временное хранилище в памяти, либо PostgreSQL для постоянного хранения данных.

---

## Содержание

1. [Особенности](#особенности)
2. [Предварительные требования](#предварительные-требования)
3. [Установка и запуск из консоли](#(Установка и запуск из консоли))
4. [Конфигурация](#конфигурация)
6. [Документация API](#документация-api)

---

## Особенности

- **Сокращение ссылок**: Генерация коротких ключей для длинных URL.
- **Перенаправление**: Перенаправление пользователей с короткого ключа на оригинальный URL.
- **Варианты хранения**:
  - Временное хранение в памяти для легковесного использования.
  - PostgreSQL для постоянного хранения.
- **Интерфейсы**:
  - HTTP API для взаимодействия через веб.
  - gRPC для высокопроизводительного клиент-серверного взаимодействия.
- **Гибкая конфигурация**: Настройка сервера и хранилища через переменные окружения.

---

## Предварительные требования

Перед запуском приложения убедитесь, что у вас установлены следующие компоненты:

- **Go**: Рекомендуемая версия 1.23.1
- **PostgreSQL**: Требуется для использования в качестве базы данных в постоянном хранилище
- **gRPC**: Требуется,для взаимодействия с gRPC интерфейсом.

---

## Установка и запуск из консоли

1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/iopuiuyhgo/OZON_test.git
   cd OZON_test
   ```

2. Установите зависимости:
   ```bash
   go mod download
   ```

3. Соберите приложение:
   ```bash
   go build -o OZON_test
   ```

4. Запустите сервер:
   ```bash
   ./OZON_test
   ```

5. Проверьте, что сервер запущен:
   - Для HTTP: Откройте `http://<SERVER_IP>:<SERVER_PORT>/page` в браузере.
   - Для gRPC: Используйте gRPC клиент для взаимодействия с сервером.

---
 
## Установка и запуск для докер контейнера

1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/iopuiuyhgo/OZON_test.git
   cd OZON_test
   ```

2. Соберите образ
   ```bash
   docker build -t ozon_test .
   ```

3. Запустите с указанием переменных окружения:
   ```bash
   docker run -e SERVER_IP=localhost -e SERVER_PORT=8080 -e USE_IN_MEMORY=false -e POSTGRES_PATH=postgres://user:password@localhost:5432/dbname -e TABLE_NAME=links -e GRPC=true -p 8080:8080 ozon_test
   ```

4. Проверьте, что сервер запущен:
   - Для HTTP: Откройте `http://<SERVER_IP>:<SERVER_PORT>/page` в браузере.
   - Для gRPC: Используйте gRPC клиент для взаимодействия с сервером.

---
 
## Конфигурация

Приложение использует переменные окружения для настройки. Ниже приведены доступные параметры:

| Имя переменной      | Описание                                   | Значение по умолчанию       |
|--------------------|-------------------------------------------|-----------------------------|
| `SERVER_IP`        | IP-адрес сервера                          | `localhost`                 |
| `SERVER_PORT`      | Порт сервера                              | `8080`                      |
| `USE_IN_MEMORY`    | Использовать временное хранилище (`true` или `false`) | `true`              |
| `POSTGRES_PATH`    | Строка подключения к PostgreSQL            |                      |
| `TABLE_NAME`       | Название таблицы в PostgreSQL              |                     |
| `GRPC`             | Включить gRPC интерфейс (`true` или `false`) | `true`              |

### Пример конфигурации

Чтобы запустить приложение с использованием PostgreSQL и gRPC:
```bash
export SERVER_IP="localhost"
export SERVER_PORT="8080"
export USE_IN_MEMORY="false"
export POSTGRES_PATH="postgres://user:password@localhost:5432/dbname"
export TABLE_NAME="links"
export GRPC="true"
```

---

## Документация API

### HTTP API

#### 1. Сократить URL (POST `/`)

- **Тело запроса**:
  ```json
  {
    "url": "https://example.com"
  }
  ```

- **Ответ**:
  ```json
  {
    "message": "Данные успешно получены",
    "URL": "http://<SERVER_IP>:<SERVER_PORT>/<короткий_ключ>"
  }
  ```

#### 2. Перенаправление на оригинальный URL (GET `/<короткий_ключ>`)

- **Ответ**: Перенаправляет на оригинальный URL.

#### 3. Просмотр веб-страницы (GET `/page`)

- **Ответ**: Отображает HTML страницу для взаимодействия с сервисом.

---

### gRPC API

#### 1. Генерация короткого ключа

- **Запрос**:
  ```proto
  message GenerateKeyRequest {
    string url = 1;
  }
  ```

- **Ответ**:
  ```proto
  message GenerateKeyResponse {
    string message = 1;
    string short_url = 2;
  }
  ```

#### 2. Перенаправление по короткому ключу

- **Запрос**:
  ```proto
  message RedirectRequest {
    string key = 1;
  }
  ```

- **Ответ**:
  ```proto
  message RedirectResponse {
    string url = 1;
  }
  ```
