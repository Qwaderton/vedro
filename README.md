# vedro

**vedro** — это простое, самодостаточное S3-подобное хранилище, написанное на Go.
Заточено под локальные нужды, не требует зависимости от облаков или баз данных.

⚠️ **Не рекомендуется для огромных проектов**, но отлично подойдёт для:
- Локального хранения артефактов
- Простейшего CI/CD
- Отладки или локальных разработок
- Временных S3-заглушек

---

## Возможности

- Эмуляция S3-интерфейса (ListBucket, GetObject)
- Поддержка ETag, Last-Modified и If-Modified-Since
- XML-ответы по стандарту S3
- Простой HTTP-сервер
- Защита от path traversal
- Middleware-логгирование и recover

---

## Конфигурация

Все настройки задаются через файл [`config.go`](./vedro/config/config.go):

```go
const (
    ServerAddr     = ":8080"      // адрес и порт сервера
    RootPath       = "/var/vedra" // корневая директория с "бакетами"
    EnableRecover = true          // включить автоматическое восстановление
    EnableLogging = true          // включить логирование запросов в stdout
)
```

---

## Как использовать

### 1. Собрать

```bash
go build -o vedrod .
```

### 2. Подготовить директорию

Создай структуру вида:

```
/var/vedra/
├── bucket1/
│   ├── file1.txt
│   └── image.png
└── bucket2/
    └── report.pdf
```

### 3. Запустить

```bash
./vedrod
```

---

## Примеры запросов

### Получить список объектов (S3 ListBucket)

```
GET http://localhost:8080/bucket1
```

Ответ (XML):

```xml
<ListBucketResult>
  <Name>bucket1</Name>
  <Contents>
    <Key>file1.txt</Key>
    <LastModified>2025-04-18T17:42:18Z</LastModified>
    <Size>123</Size>
    <ETag>...</ETag>
  </Contents>
</ListBucketResult>
```

### Скачать объект (S3 GetObject)

```
GET http://localhost:8080/bucket1/file1.txt
```

---

## Заметки

- Не поддерживает Multipart Upload, ACL, PUT/DELETE, подписанные URL и пр.
- Подразумевается, что все данные доступны локально на диске.
- Не проверяет MIME-типы вручную — использует `http.DetectContentType`.
- Ничего не кеширует — читает прямо с диска

---

## Лицензия

[Qwaderton License 1.1](LICENSE)