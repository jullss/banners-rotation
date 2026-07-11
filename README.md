# Banners Rotation

Сервис ротации баннеров на основе алгоритма Multi-Armed Bandit (UCB1).

## Запуск

```bash
cp .env.example .env  # задать POSTGRES_PASSWORD
make run
```

Сервис поднимается на `http://localhost:8080`.

## API

| Метод | URL | Описание |
|---|---|---|
| `POST` | `/slots/{slot_id}/banners/{banner_id}` | Добавить баннер в слот |
| `DELETE` | `/slots/{slot_id}/banners/{banner_id}` | Убрать баннер из слота |
| `GET` | `/slots/{slot_id}/choose?group_id=N` | Выбрать баннер (UCB1) |
| `POST` | `/slots/{slot_id}/banners/{banner_id}/click?group_id=N` | Зафиксировать клик |

## Тесты

```bash
make test                # unit-тесты
make test-integration    # интеграционные тесты (требуется Docker)
```

## Линтер

```bash
make lint
```
