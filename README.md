# Orchestrator

Распределённый оркестратор Docker-контейнеров, написанный на Go. Менеджер принимает задачи от пользователя, планирует их на воркер-ноды и следит за их состоянием.


**Manager** — принимает задачи, выбирает воркер через планировщик, следит за здоровьем задач и перезапускает упавшие.

**Worker** — запускает и останавливает Docker-контейнеры, собирает метрики ноды.

**Scheduler** — выбирает оптимальный воркер. Доступны два алгоритма:
- `roundrobin` — равномерное распределение по кругу
- `epvm` — на основе реальной загрузки CPU и памяти (EPVM-алгоритм)

## Требования

- Go 1.24+
- Docker Engine
- Пользователь добавлен в группу `docker`:
  ```bash
  sudo usermod -aG docker $USER && newgrp docker
  ```

## Установка

```bash
git clone <repo>
cd orchestrator
go mod tidy
go build -o orchestrator .
```

## Запуск

### Запуск воркера

```bash
./orchestrator worker \
  --host 0.0.0.0 \
  --port 5556 \
  --name worker-1 \
  --dbtype persistent \
  --datadir ./data
```

| Флаг        | Короткий | По умолчанию | Описание                              |
|-------------|----------|--------------|---------------------------------------|
| `--host`    | `-H`     | `0.0.0.0`    | Адрес для прослушивания               |
| `--port`    | `-p`     | `5556`       | Порт                                  |
| `--name`    | `-n`     | `worker-<uuid>` | Имя воркера                        |
| `--dbtype`  | `-d`     | `memory`     | Хранилище: `memory` или `persistent`  |
| `--datadir` | `-D`     | `./data`     | Директория для файлов BoltDB          |

### Запуск менеджера

```bash
./orchestrator manager \
  --host 0.0.0.0 \
  --port 5555 \
  --workers localhost:5556,localhost:5557,localhost:5558 \
  --scheduler epvm \
  --dbtype persistent \
  --datadir ./data
```

| Флаг        | Короткий | По умолчанию      | Описание                              |
|-------------|----------|-------------------|---------------------------------------|
| `--host`    | `-H`     | `0.0.0.0`         | Адрес для прослушивания               |
| `--port`    | `-p`     | `5555`            | Порт                                  |
| `--workers` | `-w`     | `localhost:5556`  | Список воркеров через запятую         |
| `--scheduler`| `-s`   | `epvm`            | Планировщик: `roundrobin` или `epvm`  |
| `--dbtype`  | `-d`     | `memory`          | Хранилище: `memory` или `persistent`  |
| `--datadir` | `-D`     | `./data`          | Директория для файлов BoltDB          |

## CLI-команды

### Запустить задачу

```bash
./orchestrator run --filename task.json --manager localhost:5555
```

Пример `task.json`:
```json
{
  "Name": "nginx",
  "Image": "nginx:latest",
  "CPU": 0.5,
  "Memory": 256,
  "Disk": 1,
  "ExposedPorts": {"80/tcp": {}},
  "PortBindings": {"80/tcp": "8080"},
  "RestartPolicy": "always",
  "HealthCheck": "/health"
}
```

### Остановить задачу

```bash
./orchestrator stop <task-uuid> --manager localhost:5555
```

### Список задач

```bash
./orchestrator status --manager localhost:5555
```

### Список нод

```bash
./orchestrator node --manager localhost:5555
```

## Хранилище

Поддерживаются два режима хранения данных:

| Режим        | Флаг           | Описание                            |
|--------------|----------------|-------------------------------------|
| `memory`     | `--dbtype memory`     | В памяти, данные теряются при рестарте |
| `persistent` | `--dbtype persistent` | BoltDB, файлы сохраняются в `--datadir` |

При использовании `persistent` создаются файлы:
- `<datadir>/<worker-name>_tasks.db` — на каждом воркере
- `<datadir>/tasks.db` и `<datadir>/events.db` — на менеджере

