# Миграция с GraphQL на REST API

## Статус реализации

| Этап | Описание | Статус |
|------|----------|--------|
| 0 (бэкенд) | REST-инфраструктура (handler, errors, pagination, router, SSE) | **ГОТОВО** |
| 0 (фронтенд) | Фронтенд инфраструктура (REST client, SSE client, TanStack Query) | **ГОТОВО** |
| 1 | Системные операции (config, system, version, stats, logs, notes) | **ГОТОВО** |
| 2 | Простые CRUD (tags, studios, color-presets, saved-filters) | **ГОТОВО** |
| 3 | Основные сущности (scenes, performers, galleries, images) | **ГОТОВО** |
| 4 | Вспомогательные сущности (groups, games, scene-markers, files, folders) | **ГОТОВО** |
| 5 | Операции (metadata, scrapers, plugins, packages, jobs) | **ГОТОВО** |
| 6 | Специальные операции (DLNA, SQL, StashBox, view-history, database) | **ГОТОВО** |
| 7 | Подписки → SSE (jobs, logs, scan complete) | **ГОТОВО** |
| 8 | Миграция плагинов (JS, Go, Python, React) | **ГОТОВО** |
| 9 | Очистка (удаление GraphQL) | **В ПРОЦЕССЕ** |

### Созданные файлы бэкенда

| Файл | Описание |
|------|----------|
| `internal/api/rest_handler.go` | Базовый REST handler, helpers, changeset translator |
| `internal/api/rest_router.go` | Главный роутер REST API с chi/v5 |
| `internal/api/rest_sse.go` | SSE broker с интеграцией в JobManager, Logger, ScanSubscribe |
| `internal/api/rest_models.go` | REST-специфичные модели |
| `internal/api/rest_stubs.go` | Заглушки для ещё не реализованных endpoint'ов |
| `internal/api/rest_config.go` | Этап 1: системные и конфигурационные операции |
| `internal/api/rest_tags.go` | Этап 2: Tags CRUD |
| `internal/api/rest_studios.go` | Этап 2: Studios CRUD |
| `internal/api/rest_color_presets.go` | Этап 2: Color Presets CRUD |
| `internal/api/rest_filters.go` | Этап 2: Saved Filters CRUD |
| `internal/api/rest_scenes.go` | Этап 3: Scenes CRUD |
| `internal/api/rest_performers.go` | Этап 3: Performers CRUD |
| `internal/api/rest_galleries.go` | Этап 3: Galleries CRUD |
| `internal/api/rest_images.go` | Этап 3: Images CRUD |
| `internal/api/rest_groups.go` | Этап 4: Groups CRUD |
| `internal/api/rest_games.go` | Этап 4: Games CRUD |
| `internal/api/rest_scene_markers.go` | Этап 4: Scene Markers CRUD |
| `internal/api/rest_files.go` | Этап 4: Files & Folders |
| `internal/api/rest_operations.go` | Этап 5: Metadata, Scrapers, Plugins, Packages, Jobs |
| `internal/api/rest_special.go` | Этап 6: DLNA, SQL, StashBox, View History, Database, External Player |
| `pkg/javascript/rest.go` | REST клиент для JS плагинов |
| `pkg/plugin/util/rest_client.go` | REST клиент для Go плагинов |
| `pkg/plugin/examples/python/stash_rest.py` | REST клиент для Python плагинов |

### Созданные файлы фронтенда

| Файл | Описание |
|------|----------|
| `ui/v2.5/src/core/rest-client.ts` | REST API клиент (fetch-based) |
| `ui/v2.5/src/core/sse-client.ts` | SSE клиент для real-time событий |
| `ui/v2.5/src/core/query-client.ts` | TanStack Query конфигурация и query keys |

### Текущее состояние

REST API работает **параллельно** с GraphQL на `/api/v1`. GraphQL остаётся
доступным на `/graphql` для обратной совместимости. Фронтенд инфраструктура
(REST client, SSE, TanStack Query) создана, но компоненты фронтенда ещё
используют Apollo Client. Для полного удаления GraphQL необходимо:

1. Перевести все компоненты фронтенда с Apollo Client на TanStack Query + REST
2. Обновить сторонние плагины для использования REST API
3. Убедиться в стабильности REST API
4. Удалить GraphQL код, зависимости и generated files

---

## Оглавление

1. [Обзор текущей архитектуры](#1-обзор-текущей-архитектуры)
2. [Масштаб задачи](#2-масштаб-задачи)
3. [Целевая архитектура](#3-целевая-архитектура)
4. [Этапы миграции](#4-этапы-миграции)
5. [Детальный план по этапам](#5-детальный-план-по-этапам)
6. [Структура REST API](#6-структура-rest-api)
7. [Замена подписок (Subscriptions)](#7-замена-подписок-subscriptions)
8. [Миграция фронтенда](#8-миграция-фронтенда)
9. [Миграция плагинов](#9-миграция-плагинов)
10. [Что НЕ мигрируем](#10-что-не-мигрируем)
11. [Риски и стратегия отката](#11-риски-и-стратегия-отката)

---

## 1. Обзор текущей архитектуры

### Бэкенд (Go)

- **Роутер**: `chi/v5` — уже используется для REST-маршрутов
- **GraphQL**: `gqlgen v0.17.73` — основной API для CRUD-операций
- **БД**: SQLite (через `sqlx`, без ORM)
- **Паттерн доступа к данным**: Repository + Services
- **Транзакции**: `withTxn()` / `withReadTxn()` обёртки

### Фронтенд (React + TypeScript)

- **GraphQL клиент**: Apollo Client v3.8.10
- **Кодогенерация**: `@graphql-codegen/*` — генерация типов и хуков
- **Сервисный слой**: `StashService.ts` (4,252 строк) — обёртки над Apollo
- **Сгенерированные типы**: `generated-graphql.ts` (18,523 строк)

### Потребители GraphQL API

| Потребитель | Описание | Нужна миграция? |
|---|---|---|
| Web UI (Apollo Client) | Основной клиент — все CRUD через GraphQL | **ДА** |
| JS-плагины (`gql.Do()`) | Вызывают GraphQL handler напрямую | **ДА** |
| Go-плагины (`hasura/go-graphql-client`) | HTTP-клиент к `/graphql` | **ДА** |
| Python-плагины (`requests`) | POST к `/graphql` | **ДА** |
| Stash scraper | Скрапит другой Stash через GraphQL | Частично (оставить для совместимости) |
| StashBox клиент | Внешний сервис StashBox — свой GraphQL | **НЕТ** (внешний API) |

### Размеры затронутых файлов

| Файл | Строк | Описание |
|---|---|---|
| `internal/api/generated_exec.go` | 128,245 | Сгенерированная GraphQL-схема (УДАЛЯЕТСЯ) |
| `internal/api/generated_models.go` | 1,688 | Сгенерированные модели (УДАЛЯЮТСЯ) |
| `ui/v2.5/src/core/generated-graphql.ts` | 18,523 | Сгенерированные типы/хуки (ЗАМЕНЯЕТСЯ) |
| `ui/v2.5/src/core/StashService.ts` | 4,252 | Сервисный слой (ПЕРЕПИСЫВАЕТСЯ) |
| Резолверы `internal/api/resolver_*.go` | ~60 файлов | Логика запросов (ПЕРЕПИСЫВАЕТСЯ в handlers) |

---

## 2. Масштаб задачи

### GraphQL-операции для миграции

- **Queries**: ~89 операций
- **Mutations**: ~200+ операций  
- **Subscriptions**: 3 операции (jobs, logging, scan complete)
- **Фрагменты данных**: 24 файла
- **Типы/Входные типы**: 35 файлов схемы

### Основные сущности

| Сущность | Query | Mutations | Примечания |
|---|---|---|---|
| Scene | find, findAll, streams, parse, duplicates, wall, markers | create, update, bulk, destroy, O-count, play, convert, trim, merge | Самая сложная |
| Performer | find, findAll | create, update, bulk, destroy, profile images | |
| Gallery | find, findAll | create, update, bulk, destroy, O-count, images, chapters | |
| Image | find, findAll | update, bulk, destroy, O-count | |
| Studio | find, findAll | create, update, destroy | |
| Tag | find, findAll | create, update, bulk, destroy, merge | |
| Group | find, findAll | create, update, bulk, destroy, sub-groups | |
| Game | find, findAll | create, update, destroy, O-count, OMG, views | |
| SceneMarker | find | create, update, destroy | |
| Config | get | configure (general, interface, UI, defaults, scraping, DLNA, plugin) | |
| Metadata | — | scan, generate, auto-tag, clean, identify, import, export | Все возвращают job ID |
| Scraper | list, scrape* | reload | Множество scrape-вариантов |
| Plugin | list, tasks | reload, run, enable/disable | |
| Package | installed, available | install, update, uninstall | |
| Job | queue, find | stop, stopAll | |
| SavedFilter | find | save, destroy | |
| ColorPreset | find | create, update, destroy | |
| File | find | move, delete, fingerprints, scan threats | |
| DLNA | status | enable, disable, tempIP | |
| SQL | — | query, exec (dangerous) | |
| StashBox | — | submit, batch tag | |
| ViewHistory | find | — | |

---

## 3. Целевая архитектура

### Принципы

1. **REST JSON API** — для всех CRUD-операций и запросов
2. **SSE (Server-Sent Events)** — для подписок (jobs, logs, scan complete)
3. **TanStack Query (React Query)** — замена Apollo Client на фронтенде
4. **TypeScript типы** — ручные типы вместо кодогенерации (или генерация из OpenAPI)
5. **Fetch API** — вместо Apollo HTTP-клиента

### Почему именно эти технологии

| Технология | Причина выбора |
|---|---|
| REST | Простота, стандарт, отлично подходит для локального приложения |
| SSE | Проще WebSocket для однонаправленного потока (jobs, logs), нативная поддержка браузера, автореконнект |
| TanStack Query | Лёгкий, кеширование, мутации, оптимистичные обновления, не привязан к GraphQL |
| Нативный fetch | Никаких тяжёлых клиентов, прост и понятен |

### Структура URL

```
/api/v1/scenes          — CRUD сцен
/api/v1/performers      — CRUD перформеров
/api/v1/galleries       — CRUD галерей
/api/v1/images          — CRUD изображений
/api/v1/studios         — CRUD студий
/api/v1/tags            — CRUD тегов
/api/v1/groups          — CRUD групп
/api/v1/games           — CRUD игр
/api/v1/scene-markers   — CRUD маркеров
/api/v1/config          — Конфигурация
/api/v1/metadata        — Операции с метаданными
/api/v1/scrapers        — Скраперы
/api/v1/plugins         — Плагины
/api/v1/packages        — Пакеты
/api/v1/jobs            — Задачи
/api/v1/filters         — Сохранённые фильтры
/api/v1/color-presets   — Цветовые пресеты
/api/v1/files           — Файловые операции
/api/v1/dlna            — DLNA
/api/v1/system          — Системные операции
/api/v1/stash-box       — StashBox операции
/api/v1/sql             — SQL операции (dangerous)
/api/v1/events          — SSE-поток событий
```

---

## 4. Этапы миграции

### Общая стратегия: Поэтапная замена

GraphQL и REST будут работать параллельно на время миграции. Каждый этап полностью заменяет одну группу операций.

```
Этап 0: Инфраструктура REST + SSE
Этап 1: Системные операции (config, system, version, stats)
Этап 2: Простые CRUD-сущности (tags, studios, color-presets, saved-filters)
Этап 3: Основные сущности (scenes, performers, galleries, images)
Этап 4: Вспомогательные сущности (groups, games, scene-markers, files)
Этап 5: Операции (metadata, scrapers, plugins, packages, jobs)
Этап 6: Специальные операции (DLNA, SQL, StashBox, view-history)
Этап 7: Подписки → SSE
Этап 8: Миграция плагинов
Этап 9: Очистка (удаление GraphQL)
```

---

## 5. Детальный план по этапам

### Этап 0: Инфраструктура REST + SSE

#### Бэкенд

**0.1. REST-фреймворк и хелперы**

Создать базовую инфраструктуру в `internal/api/`:

```
internal/api/
├── rest/
│   ├── handler.go          — Базовый REST-хэндлер с helpers (respond, decode, error)
│   ├── middleware.go        — Middleware для REST (pagination, transactions)
│   ├── pagination.go        — Парсинг параметров пагинации из query string
│   ├── filter.go            — Парсинг фильтров из JSON body
│   ├── errors.go            — Стандартные HTTP-ошибки (400, 404, 500)
│   ├── router.go            — Регистрация всех REST-маршрутов
│   └── sse.go               — SSE-хэндлер для событий
```

**Базовый handler:**
```go
type Handler struct {
    repository models.Repository
    sceneService manager.SceneService
    // ... другие сервисы
}

// Ответ JSON
func respond(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

// Чтение JSON-тела
func decode(r *http.Request, v interface{}) error {
    return json.NewDecoder(r.Body).Decode(v)
}

// Обёртка для транзакций
func (h *Handler) withReadTxn(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) error) {
    // ...
}

func (h *Handler) withTxn(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) error) {
    // ...
}
```

**Стандартный формат ответа:**
```json
// Одиночный объект
{
  "data": { ... }
}

// Список с пагинацией
{
  "data": [ ... ],
  "count": 1234
}

// Ошибка
{
  "error": "описание ошибки",
  "code": "NOT_FOUND"
}
```

**Формат фильтрации (сохраняем совместимость с текущей системой):**
```
GET /api/v1/scenes?page=1&per_page=25&sort=date&direction=DESC
POST /api/v1/scenes/query  — для сложных фильтров (JSON body)
```

```json
// POST /api/v1/scenes/query
{
  "filter": {
    "page": 1,
    "per_page": 25,
    "sort": "date",
    "direction": "DESC",
    "q": "search term"
  },
  "scene_filter": {
    "rating100": { "value": 80, "modifier": "GREATER_THAN" },
    "tags": {
      "value": ["1", "2"],
      "modifier": "INCLUDES_ALL",
      "depth": 0
    },
    "AND": {
      "performers": { "value": ["5"], "modifier": "INCLUDES" }
    }
  }
}
```

**0.2. SSE-инфраструктура**

```go
// internal/api/rest/sse.go

type SSEBroker struct {
    clients    map[chan SSEEvent]struct{}
    register   chan chan SSEEvent
    unregister chan chan SSEEvent
    broadcast  chan SSEEvent
}

type SSEEvent struct {
    Type string      `json:"type"`
    Data interface{} `json:"data"`
}

// Типы событий:
// "job.add", "job.update", "job.remove"
// "log.entry"
// "scan.complete"
```

**0.3. Регистрация маршрутов**

```go
// internal/api/rest/router.go
func (h *Handler) Routes() chi.Router {
    r := chi.NewRouter()
    
    // Middleware для REST
    r.Use(middleware.SetHeader("Content-Type", "application/json"))
    
    r.Route("/scenes", h.sceneRoutes)
    r.Route("/performers", h.performerRoutes)
    // ... и т.д.
    
    r.Get("/events", h.sseHandler)  // SSE
    
    return r
}
```

**Монтирование в server.go:**
```go
// Существующий GraphQL (временно)
r.Handle("/graphql", gqlHandler)

// Новый REST API
r.Mount("/api/v1", restHandler.Routes())
```

#### Фронтенд

**0.4. API-клиент**

Создать `ui/v2.5/src/core/api/`:

```
src/core/api/
├── client.ts        — Базовый HTTP-клиент (fetch wrapper)
├── types.ts         — Общие типы (Pagination, Error, Response)
├── sse.ts           — SSE-клиент для событий
├── queryClient.ts   — Конфигурация TanStack Query
└── index.ts         — Re-exports
```

**Базовый клиент:**
```typescript
// src/core/api/client.ts

const BASE_URL = "/api/v1";

interface ApiResponse<T> {
  data: T;
  count?: number;
}

interface ApiError {
  error: string;
  code: string;
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  options?: RequestInit
): Promise<ApiResponse<T>> {
  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
    ...options,
  });
  
  if (!res.ok) {
    const error: ApiError = await res.json();
    throw new ApiClientError(error.error, error.code, res.status);
  }
  
  return res.json();
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
  put: <T>(path: string, body: unknown) => request<T>("PUT", path, body),
  patch: <T>(path: string, body: unknown) => request<T>("PATCH", path, body),
  delete: <T>(path: string, body?: unknown) => request<T>("DELETE", path, body),
};
```

**SSE-клиент:**
```typescript
// src/core/api/sse.ts

type EventHandler = (data: unknown) => void;

class SSEClient {
  private source: EventSource | null = null;
  private handlers = new Map<string, Set<EventHandler>>();
  
  connect() {
    this.source = new EventSource("/api/v1/events");
    this.source.onmessage = (event) => {
      const { type, data } = JSON.parse(event.data);
      this.handlers.get(type)?.forEach(h => h(data));
    };
  }
  
  on(type: string, handler: EventHandler) { ... }
  off(type: string, handler: EventHandler) { ... }
  disconnect() { ... }
}

export const sseClient = new SSEClient();
```

**0.5. Установка TanStack Query**

```bash
cd ui/v2.5 && bun add @tanstack/react-query
```

**QueryClient setup:**
```typescript
// src/core/api/queryClient.ts
import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60,        // 1 минута
      gcTime: 1000 * 60 * 5,       // 5 минут (бывший cacheTime)
      refetchOnWindowFocus: false,   // локальное приложение
      retry: 1,
    },
  },
});
```

---

### Этап 1: Системные операции

#### Бэкенд — REST handlers

**1.1. Конфигурация**
```
GET    /api/v1/config                           → configuration
PUT    /api/v1/config/general                    → configureGeneral
PUT    /api/v1/config/interface                  → configureInterface
PUT    /api/v1/config/defaults                   → configureDefaults
PUT    /api/v1/config/ui                         → configureUI
PUT    /api/v1/config/ui/setting                 → configureUISetting
PUT    /api/v1/config/scraping                   → configureScraping
PUT    /api/v1/config/dlna                       → configureDLNA
PUT    /api/v1/config/plugin/:pluginId           → configurePlugin
POST   /api/v1/config/api-key                    → generateAPIKey
```

**1.2. Система**
```
GET    /api/v1/system/status                     → systemStatus
GET    /api/v1/system/version                    → version
GET    /api/v1/system/latest-version             → latestversion
GET    /api/v1/system/directory?path=...         → directory
POST   /api/v1/system/setup                      → setup
POST   /api/v1/system/migrate                    → migrate
POST   /api/v1/system/download-ffmpeg            → downloadFFMpeg
POST   /api/v1/system/validate-stashbox          → validateStashBoxCredentials
```

**1.3. Статистика**
```
GET    /api/v1/stats                             → stats
GET    /api/v1/stats/o-count                     → oCountStats
```

**1.4. Логи**
```
GET    /api/v1/logs                              → logs
```

**1.5. Заметки**
```
GET    /api/v1/notes                             → readNotesFile
PUT    /api/v1/notes                             → writeNotesFile
```

#### Фронтенд — hooks + types

Создать:
```
src/core/api/
├── config.ts          — типы + хуки для конфигурации
├── system.ts          — типы + хуки для системы
├── stats.ts           — типы + хуки для статистики
```

**Пример хука:**
```typescript
// src/core/api/config.ts
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";

export const configKeys = {
  all: ["config"] as const,
};

export function useConfiguration() {
  return useQuery({
    queryKey: configKeys.all,
    queryFn: () => api.get<ConfigResult>("/config"),
  });
}

export function useConfigureGeneral() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ConfigGeneralInput) => api.put<ConfigGeneralResult>("/config/general", input),
    onSuccess: () => qc.invalidateQueries({ queryKey: configKeys.all }),
  });
}
```

**Замена в компонентах:**
```typescript
// Было:
const { data } = GQL.useConfigurationQuery();

// Стало:
const { data } = useConfiguration();
```

---

### Этап 2: Простые CRUD-сущности

#### 2.1. Tags
```
GET    /api/v1/tags/:id                          → findTag
POST   /api/v1/tags/query                        → findTags (с фильтрами)
GET    /api/v1/tags/colors                        → findTagColors
POST   /api/v1/tags                              → tagCreate
PUT    /api/v1/tags/:id                          → tagUpdate
PUT    /api/v1/tags/bulk                         → bulkTagUpdate
DELETE /api/v1/tags/:id                          → tagDestroy
DELETE /api/v1/tags                              → tagsDestroy (body: {ids: [...]})
POST   /api/v1/tags/merge                        → tagsMerge
```

#### 2.2. Studios
```
GET    /api/v1/studios/:id                       → findStudio
POST   /api/v1/studios/query                     → findStudios
POST   /api/v1/studios                           → studioCreate
PUT    /api/v1/studios/:id                       → studioUpdate
DELETE /api/v1/studios/:id                       → studioDestroy
DELETE /api/v1/studios                           → studiosDestroy
```

#### 2.3. Color Presets
```
GET    /api/v1/color-presets/:id                 → findColorPreset
POST   /api/v1/color-presets/query               → findColorPresets
POST   /api/v1/color-presets                     → colorPresetCreate
PUT    /api/v1/color-presets/:id                 → colorPresetUpdate
DELETE /api/v1/color-presets/:id                 → colorPresetDestroy
```

#### 2.4. Saved Filters
```
GET    /api/v1/filters/:id                       → findSavedFilter
GET    /api/v1/filters?mode=...                  → findSavedFilters
POST   /api/v1/filters                           → saveFilter
DELETE /api/v1/filters/:id                       → destroySavedFilter
```

#### Паттерн реализации handler'а (Go)

```go
// internal/api/rest/handler_tags.go

func (h *Handler) tagRoutes(r chi.Router) {
    r.Get("/{id}", h.findTag)
    r.Post("/query", h.findTags)
    r.Get("/colors", h.findTagColors)
    r.Post("/", h.createTag)
    r.Put("/{id}", h.updateTag)
    r.Put("/bulk", h.bulkUpdateTags)
    r.Delete("/{id}", h.destroyTag)
    r.Delete("/", h.destroyTags)
    r.Post("/merge", h.mergeTags)
}

func (h *Handler) findTag(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.Atoi(chi.URLParam(r, "id"))
    
    var tag *models.Tag
    if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
        var err error
        tag, err = h.repository.Tag.Find(ctx, id)
        return err
    }); err != nil {
        respondError(w, err)
        return
    }
    
    if tag == nil {
        respondNotFound(w)
        return
    }
    
    respond(w, http.StatusOK, tag)
}

func (h *Handler) findTags(w http.ResponseWriter, r *http.Request) {
    var input struct {
        Filter     *models.FindFilterType `json:"filter"`
        TagFilter  *models.TagFilterType  `json:"tag_filter"`
    }
    if err := decode(r, &input); err != nil {
        respondBadRequest(w, err)
        return
    }
    
    var result *models.FindTagsResultType
    if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
        var err error
        result, err = h.repository.Tag.Query(ctx, input.TagFilter, input.Filter)
        return err
    }); err != nil {
        respondError(w, err)
        return
    }
    
    respond(w, http.StatusOK, result)
}
```

#### Паттерн фронтенда

```typescript
// src/core/api/tags.ts

export interface Tag {
  id: string;
  name: string;
  // ...все поля
}

export const tagKeys = {
  all: ["tags"] as const,
  lists: () => [...tagKeys.all, "list"] as const,
  list: (filter: FindFilterType, tagFilter?: TagFilterType) => 
    [...tagKeys.lists(), { filter, tagFilter }] as const,
  details: () => [...tagKeys.all, "detail"] as const,
  detail: (id: string) => [...tagKeys.details(), id] as const,
};

export function useFindTag(id: string) {
  return useQuery({
    queryKey: tagKeys.detail(id),
    queryFn: () => api.get<Tag>(`/tags/${id}`),
    enabled: !!id,
  });
}

export function useFindTags(filter: FindFilterType, tagFilter?: TagFilterType) {
  return useQuery({
    queryKey: tagKeys.list(filter, tagFilter),
    queryFn: () => api.post<FindTagsResult>("/tags/query", { filter, tag_filter: tagFilter }),
  });
}

export function useTagCreate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: TagCreateInput) => api.post<Tag>("/tags", input),
    onSuccess: () => qc.invalidateQueries({ queryKey: tagKeys.lists() }),
  });
}

export function useTagUpdate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: TagUpdateInput) => api.put<Tag>(`/tags/${input.id}`, input),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: tagKeys.detail(vars.id) });
      qc.invalidateQueries({ queryKey: tagKeys.lists() });
    },
  });
}
```

---

### Этап 3: Основные сущности

#### 3.1. Scenes (самая сложная сущность)

```
# CRUD
GET    /api/v1/scenes/:id                        → findScene
POST   /api/v1/scenes/query                      → findScenes
POST   /api/v1/scenes                            → sceneCreate
PUT    /api/v1/scenes/:id                        → sceneUpdate
PUT    /api/v1/scenes/bulk                       → bulkSceneUpdate
PUT    /api/v1/scenes/batch                      → scenesUpdate (массовое обновление)
DELETE /api/v1/scenes/:id                        → sceneDestroy
DELETE /api/v1/scenes                            → scenesDestroy

# Специальные запросы
GET    /api/v1/scenes/:id/streams                → sceneStreams
GET    /api/v1/scenes/by-hash                    → findSceneByHash
POST   /api/v1/scenes/duplicates                 → findDuplicateScenes
POST   /api/v1/scenes/parse-filenames            → parseSceneFilenames
GET    /api/v1/scenes/wall?q=...                 → sceneWall

# O-Count
POST   /api/v1/scenes/:id/o                      → sceneAddO
DELETE /api/v1/scenes/:id/o                      → sceneDeleteO
POST   /api/v1/scenes/:id/o/reset                → sceneResetO

# OMG-Count
POST   /api/v1/scenes/:id/omg                    → sceneAddOmg
DELETE /api/v1/scenes/:id/omg                    → sceneDeleteOmg
POST   /api/v1/scenes/:id/omg/reset              → sceneResetOmg

# Play count
POST   /api/v1/scenes/:id/play                   → sceneAddPlay
DELETE /api/v1/scenes/:id/play                   → sceneDeletePlay
POST   /api/v1/scenes/:id/play/reset             → sceneResetPlayCount

# Activity
PUT    /api/v1/scenes/:id/activity               → sceneSaveActivity
POST   /api/v1/scenes/:id/activity/reset         → sceneResetActivity

# Video operations
POST   /api/v1/scenes/:id/convert/mp4            → sceneConvertToMp4
POST   /api/v1/scenes/:id/convert/hls            → sceneConvertHLSToMP4
POST   /api/v1/scenes/:id/reduce-resolution      → sceneReduceResolution
POST   /api/v1/scenes/:id/trim                   → sceneTrimVideo
POST   /api/v1/scenes/:id/regenerate-sprites     → sceneRegenerateSprites
PUT    /api/v1/scenes/:id/broken                 → sceneSetBroken / sceneSetNotBroken
POST   /api/v1/scenes/:id/screenshot             → sceneGenerateScreenshot
POST   /api/v1/scenes/:id/filtered-screenshot    → sceneSaveFilteredScreenshot

# Merge & files
POST   /api/v1/scenes/:id/merge                  → sceneMerge
PUT    /api/v1/scenes/:id/primary-file           → sceneSetPrimaryFile
PUT    /api/v1/scenes/:id/assign-file            → sceneAssignFile

# Similarity
POST   /api/v1/scenes/:id/recalculate-similarity → recalculateSceneSimilarities
```

#### 3.2. Performers
```
GET    /api/v1/performers/:id                    → findPerformer
POST   /api/v1/performers/query                  → findPerformers
POST   /api/v1/performers                        → performerCreate
PUT    /api/v1/performers/:id                    → performerUpdate
PUT    /api/v1/performers/bulk                   → bulkPerformerUpdate
DELETE /api/v1/performers/:id                    → performerDestroy
DELETE /api/v1/performers                        → performersDestroy

# Profile images
POST   /api/v1/performers/:id/profile-images     → performerProfileImageCreate
PUT    /api/v1/performers/:id/profile-images/:imageId → performerProfileImageUpdate
DELETE /api/v1/performers/:id/profile-images/:imageId → performerProfileImageDestroy
```

#### 3.3. Galleries
```
GET    /api/v1/galleries/:id                     → findGallery
POST   /api/v1/galleries/query                   → findGalleries
POST   /api/v1/galleries                         → galleryCreate
PUT    /api/v1/galleries/:id                     → galleryUpdate
PUT    /api/v1/galleries/bulk                    → bulkGalleryUpdate
DELETE /api/v1/galleries/:id                     → galleryDestroy

# O-Count, OMG, Play
POST   /api/v1/galleries/:id/o                   → galleryAddO
DELETE /api/v1/galleries/:id/o                   → galleryDeleteO
POST   /api/v1/galleries/:id/o/reset             → galleryResetO
POST   /api/v1/galleries/:id/omg                 → galleryAddOMG (аналогично)
POST   /api/v1/galleries/:id/play                → galleryAddPlay (аналогично)

# Images
POST   /api/v1/galleries/:id/images              → addGalleryImages
DELETE /api/v1/galleries/:id/images              → removeGalleryImages
PUT    /api/v1/galleries/:id/cover               → setGalleryCover
DELETE /api/v1/galleries/:id/cover               → resetGalleryCover
PUT    /api/v1/galleries/:id/primary-file        → gallerySetPrimaryFile

# Chapters
POST   /api/v1/galleries/:id/chapters            → galleryChapterCreate
PUT    /api/v1/galleries/:id/chapters/:chapterId → galleryChapterUpdate
DELETE /api/v1/galleries/:id/chapters/:chapterId → galleryChapterDestroy
```

#### 3.4. Images
```
GET    /api/v1/images/:id                        → findImage
POST   /api/v1/images/query                      → findImages
PUT    /api/v1/images/:id                        → imageUpdate
PUT    /api/v1/images/bulk                       → bulkImageUpdate
DELETE /api/v1/images/:id                        → imageDestroy
DELETE /api/v1/images                            → imagesDestroy

# O-Count, OMG
POST   /api/v1/images/:id/o                      → imageIncrementO / addO
DELETE /api/v1/images/:id/o                      → imageDecrementO / deleteO
POST   /api/v1/images/:id/o/reset                → imageResetO
POST   /api/v1/images/:id/omg                    → (аналогично)
PUT    /api/v1/images/:id/primary-file           → imageSetPrimaryFile
```

---

### Этап 4: Вспомогательные сущности

#### 4.1. Groups
```
GET    /api/v1/groups/:id                        → findGroup
POST   /api/v1/groups/query                      → findGroups
POST   /api/v1/groups                            → groupCreate
PUT    /api/v1/groups/:id                        → groupUpdate
PUT    /api/v1/groups/bulk                       → bulkGroupUpdate
DELETE /api/v1/groups/:id                        → groupDestroy
DELETE /api/v1/groups                            → groupsDestroy

# Sub-groups
POST   /api/v1/groups/:id/sub-groups             → addGroupSubGroups
DELETE /api/v1/groups/:id/sub-groups             → removeGroupSubGroups
PUT    /api/v1/groups/:id/sub-groups/reorder     → reorderSubGroups
```

#### 4.2. Games
```
GET    /api/v1/games/:id                         → findGame
POST   /api/v1/games/query                       → findGames
POST   /api/v1/games                             → gameCreate
PUT    /api/v1/games/:id                         → gameUpdate
DELETE /api/v1/games/:id                         → gameDestroy

# O/OMG/Views (как у Scenes)
POST   /api/v1/games/:id/o                       → gameAddO
POST   /api/v1/games/:id/omg                     → gameAddOmg
POST   /api/v1/games/:id/view                    → gameAddView
# ... delete, reset аналогично
```

#### 4.3. Scene Markers
```
POST   /api/v1/scene-markers/query               → findSceneMarkers
GET    /api/v1/scene-markers/wall?q=...          → markerWall
GET    /api/v1/scene-markers/strings?q=...&sort=... → markerStrings
GET    /api/v1/scene-markers/tags/:sceneId       → sceneMarkerTags
POST   /api/v1/scene-markers                     → sceneMarkerCreate
PUT    /api/v1/scene-markers/:id                 → sceneMarkerUpdate
DELETE /api/v1/scene-markers/:id                 → sceneMarkerDestroy
DELETE /api/v1/scene-markers                     → sceneMarkersDestroy
```

#### 4.4. Files
```
GET    /api/v1/files/:id                         → findFile
POST   /api/v1/files/query                       → findFiles
POST   /api/v1/files/move                        → moveFiles
DELETE /api/v1/files                             → deleteFiles
PUT    /api/v1/files/fingerprints                → fileSetFingerprints
POST   /api/v1/files/:id/scan-threats            → scanVideoFileThreats
POST   /api/v1/files/scan-all-threats            → scanAllScenesForThreats

GET    /api/v1/folders/:id                       → findFolder
POST   /api/v1/folders/query                     → findFolders
```

---

### Этап 5: Операции

#### 5.1. Metadata
```
POST   /api/v1/metadata/scan                     → metadataScan
POST   /api/v1/metadata/generate                 → metadataGenerate
POST   /api/v1/metadata/auto-tag                 → metadataAutoTag
POST   /api/v1/metadata/clean                    → metadataClean
POST   /api/v1/metadata/clean-generated          → metadataCleanGenerated
POST   /api/v1/metadata/identify                 → metadataIdentify
POST   /api/v1/metadata/export                   → metadataExport
POST   /api/v1/metadata/import                   → metadataImport
POST   /api/v1/metadata/export-objects            → exportObjects
POST   /api/v1/metadata/import-objects            → importObjects
```

#### 5.2. Scrapers
```
GET    /api/v1/scrapers?types=...                → listScrapers
POST   /api/v1/scrapers/reload                   → reloadScrapers
POST   /api/v1/scrapers/scene                    → scrapeSingleScene
POST   /api/v1/scrapers/scenes                   → scrapeMultiScenes
POST   /api/v1/scrapers/performer                → scrapeSinglePerformer
POST   /api/v1/scrapers/performers               → scrapeMultiPerformers
POST   /api/v1/scrapers/gallery                  → scrapeSingleGallery
POST   /api/v1/scrapers/group                    → scrapeSingleGroup
POST   /api/v1/scrapers/image                    → scrapeSingleImage
POST   /api/v1/scrapers/url                      → scrapeURL
```

#### 5.3. Plugins
```
GET    /api/v1/plugins                           → plugins
GET    /api/v1/plugins/tasks                     → pluginTasks
POST   /api/v1/plugins/reload                    → reloadPlugins
PUT    /api/v1/plugins/enabled                   → setPluginsEnabled
POST   /api/v1/plugins/:id/run                   → runPluginTask
POST   /api/v1/plugins/:id/operation             → runPluginOperation
```

#### 5.4. Packages
```
GET    /api/v1/packages/installed?type=...       → installedPackages
GET    /api/v1/packages/available?type=...&source=... → availablePackages
POST   /api/v1/packages/install                  → installPackages
POST   /api/v1/packages/update                   → updatePackages
POST   /api/v1/packages/uninstall                → uninstallPackages
```

#### 5.5. Jobs
```
GET    /api/v1/jobs                              → jobQueue
GET    /api/v1/jobs/:id                          → findJob
POST   /api/v1/jobs/:id/stop                     → stopJob
POST   /api/v1/jobs/stop-all                     → stopAllJobs
```

---

### Этап 6: Специальные операции

#### 6.1. DLNA
```
GET    /api/v1/dlna/status                       → dlnaStatus
POST   /api/v1/dlna/enable                       → enableDLNA
POST   /api/v1/dlna/disable                      → disableDLNA
POST   /api/v1/dlna/temp-ip                      → addTempDLNAIP
DELETE /api/v1/dlna/temp-ip                      → removeTempDLNAIP
```

#### 6.2. SQL (dangerous)
```
POST   /api/v1/sql/query                         → querySQL
POST   /api/v1/sql/exec                          → execSQL
```

#### 6.3. StashBox
```
POST   /api/v1/stash-box/fingerprints            → submitStashBoxFingerprints
POST   /api/v1/stash-box/scene-draft             → submitStashBoxSceneDraft
POST   /api/v1/stash-box/performer-draft         → submitStashBoxPerformerDraft
POST   /api/v1/stash-box/batch/performers        → stashBoxBatchPerformerTag
POST   /api/v1/stash-box/batch/studios           → stashBoxBatchStudioTag
```

#### 6.4. View History
```
POST   /api/v1/view-history/query                → findViewHistory
```

#### 6.5. Database
```
POST   /api/v1/database/backup                   → backupDatabase
POST   /api/v1/database/anonymise                → anonymiseDatabase
POST   /api/v1/database/optimise                 → optimiseDatabase
POST   /api/v1/database/migrate-hash-naming      → migrateHashNaming
POST   /api/v1/database/migrate-screenshots      → migrateSceneScreenshots
POST   /api/v1/database/migrate-blobs            → migrateBlobs
```

#### 6.6. Misc
```
POST   /api/v1/misc/open-external-player/:id     → openInExternalPlayer
```

---

### Этап 7: Подписки → SSE

#### Бэкенд

Создать единый SSE-эндпоинт `GET /api/v1/events`:

```go
func (h *Handler) sseHandler(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    events := h.sseBroker.Subscribe()
    defer h.sseBroker.Unsubscribe(events)
    
    for {
        select {
        case event := <-events:
            data, _ := json.Marshal(event)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

**Типы событий SSE:**
```
event: job.add
data: {"job": {...}}

event: job.update
data: {"job": {...}}

event: job.remove
data: {"job": {...}}

event: log.entries
data: [{"time": "...", "level": "info", "message": "..."}]

event: scan.complete
data: {}
```

#### Фронтенд

```typescript
// src/core/api/useSSE.ts
import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

export function useSSEEvents() {
  const qc = useQueryClient();
  
  useEffect(() => {
    const source = new EventSource("/api/v1/events");
    
    source.addEventListener("job.add", (e) => {
      qc.invalidateQueries({ queryKey: ["jobs"] });
    });
    
    source.addEventListener("job.update", (e) => {
      qc.invalidateQueries({ queryKey: ["jobs"] });
    });
    
    source.addEventListener("scan.complete", () => {
      // Инвалидируем все запросы (как resetStore в Apollo)
      qc.invalidateQueries();
    });
    
    return () => source.close();
  }, [qc]);
}

// Для подписки на логи
export function useLogStream(onLog: (entries: LogEntry[]) => void) {
  useEffect(() => {
    const source = new EventSource("/api/v1/events");
    source.addEventListener("log.entries", (e) => {
      onLog(JSON.parse(e.data));
    });
    return () => source.close();
  }, [onLog]);
}
```

---

### Этап 8: Миграция плагинов

#### 8.1. JavaScript-плагины

Заменить `gql.Do(query, variables)` на REST-вызовы:

```javascript
// Было:
var result = gql.Do(`query { findTag(id: $id) { id name } }`, { id: "1" });

// Стало:
var result = api.get("/api/v1/tags/1");
```

**Реализация `api` объекта для JS-плагинов:**
```go
// pkg/javascript/api.go
type API struct {
    handler http.Handler
}

func (a *API) Get(path string) interface{} {
    // Вызывает REST handler напрямую (без сети)
}

func (a *API) Post(path string, body interface{}) interface{} { ... }
func (a *API) Put(path string, body interface{}) interface{} { ... }
func (a *API) Delete(path string, body interface{}) interface{} { ... }
```

#### 8.2. Go-плагины

Заменить `graphql.Client` на HTTP-клиент:

```go
// Было:
client := util.NewClient(serverURL)
var query struct { FindTag struct { ID string; Name string } `graphql:"findTag(id: $id)"` }
client.Query(ctx, &query, map[string]interface{}{"id": "1"})

// Стало:
client := util.NewRESTClient(serverURL)
tag, err := client.Get("/api/v1/tags/1")
```

#### 8.3. Python-плагины

Обновить `StashInterface`:

```python
# Было:
def findTagIdWithName(self, name):
    query = """query { findTags(tag_filter: {name: {value: $name}}) { tags { id } } }"""
    result = self._callGraphQL(query, {"name": name})
    
# Стало:
def findTagIdWithName(self, name):
    result = self._post("/api/v1/tags/query", {
        "tag_filter": {"name": {"value": name, "modifier": "EQUALS"}}
    })
```

---

### Этап 9: Очистка

#### Удалить с бэкенда:
- `graphql/schema/` — все GraphQL-схемы
- `internal/api/generated_exec.go` (128,245 строк)
- `internal/api/generated_models.go` (1,688 строк)
- `internal/api/resolver_*.go` — все резолверы (~60 файлов)
- `internal/api/loaders/` — DataLoaders (больше не нужны)
- `gqlgen.yml` — конфиг кодогенерации

#### Удалить зависимости Go:
- `github.com/99designs/gqlgen`
- `github.com/vektah/dataloaden`
- `github.com/vektah/gqlparser/v2`
- `github.com/hasura/go-graphql-client` (если плагины переведены)

#### Удалить с фронтенда:
- `ui/v2.5/graphql/` — все .graphql файлы (68 файлов)
- `ui/v2.5/src/core/generated-graphql.ts` (18,523 строк)
- `ui/v2.5/src/core/createClient.ts` — Apollo Client setup
- `ui/v2.5/codegen.ts` — конфиг кодогенерации

#### Удалить npm-зависимости:
- `@apollo/client`
- `graphql`
- `graphql-tag`
- `graphql-ws`
- `apollo-upload-client`
- `@graphql-codegen/*` (5 пакетов)
- `@types/apollo-upload-client`

#### Удалить из Makefile:
- `generate-backend` (gqlgen generate)
- `generate-ui` (graphql-codegen)
- Обновить `generate` target

#### Обновить:
- `StashService.ts` → полностью переписать или удалить (заменён на api/*.ts)
- `README.md` — обновить информацию об API
- `docs/DEVELOPMENT.md` — обновить инструкции разработки

---

## 6. Структура REST API

### Конвенции

| Аспект | Правило |
|---|---|
| Формат URL | `/api/v1/{resource}/{id?}/{sub-resource?}` |
| HTTP-методы | GET (чтение), POST (создание/действие), PUT (обновление), DELETE (удаление) |
| Сложные фильтры | `POST /api/v1/{resource}/query` с JSON body |
| Простые списки | `GET /api/v1/{resource}?page=1&per_page=25&sort=name&direction=ASC` |
| Формат ответа | `{"data": ..., "count": N}` для списков, `{"data": ...}` для одиночных |
| Ошибки | `{"error": "message", "code": "ERROR_CODE"}` |
| Коды статусов | 200 OK, 201 Created, 204 No Content, 400 Bad Request, 404 Not Found, 500 Server Error |
| Content-Type | `application/json` везде |
| File uploads | `multipart/form-data` (для скриншотов, изображений) |

### Формат пагинации

```json
// Request
{
  "filter": {
    "page": 1,
    "per_page": 25,
    "sort": "date",
    "direction": "DESC",
    "q": "search term"
  }
}

// Response
{
  "data": [...],
  "count": 1234
}
```

### Формат фильтров

Сохраняем полную совместимость с текущей системой фильтрации GraphQL, но передаём через JSON body:

```json
{
  "filter": { "page": 1, "per_page": 25, "sort": "date", "direction": "DESC" },
  "scene_filter": {
    "rating100": { "value": 80, "modifier": "GREATER_THAN" },
    "tags": { "value": ["1", "2"], "modifier": "INCLUDES_ALL", "depth": 0 },
    "performers": { "value": ["5"], "modifier": "INCLUDES" },
    "AND": {
      "studios": { "value": ["3"], "modifier": "INCLUDES" }
    },
    "OR": {
      "title": { "value": "test", "modifier": "MATCHES_REGEX" }
    }
  }
}
```

---

## 7. Замена подписок (Subscriptions)

### Текущие подписки

| Подписка | Используется для | Замена |
|---|---|---|
| `jobsSubscribe` | Обновления статуса задач | SSE: `job.add`, `job.update`, `job.remove` |
| `loggingSubscribe` | Потоковая передача логов | SSE: `log.entries` |
| `scanCompleteSubscribe` | Уведомление о завершении скана | SSE: `scan.complete` |

### Почему SSE, а не WebSocket

1. **Однонаправленный поток** — все подписки передают данные только от сервера к клиенту
2. **Автоматический реконнект** — EventSource автоматически переподключается
3. **Простота** — не нужен специальный протокол (WebSocket требует handshake)
4. **Нативная поддержка** — работает через стандартный HTTP, проходит через прокси
5. **Меньше кода** — не нужна библиотека `graphql-ws`

### Реализация

Единый SSE-эндпоинт: `GET /api/v1/events`

Клиент подключается один раз и получает все события. Фильтрация по типу события на клиенте.

---

## 8. Миграция фронтенда

### Стратегия

1. **Параллельная работа**: Apollo и TanStack Query работают одновременно
2. **Покомпонентная миграция**: каждый компонент переводится отдельно
3. **Типы**: создаются вручную на основе существующих GraphQL-типов (из `generated-graphql.ts`)

### Структура новых файлов

```
src/core/api/
├── client.ts           — HTTP-клиент (fetch wrapper)
├── sse.ts              — SSE-клиент
├── queryClient.ts      — TanStack Query конфигурация
├── types.ts            — Общие типы (Pagination, Error)
├── types/
│   ├── scene.ts        — Типы для сцен
│   ├── performer.ts    — Типы для перформеров
│   ├── gallery.ts      — Типы для галерей
│   ├── image.ts        — Типы для изображений
│   ├── studio.ts       — Типы для студий
│   ├── tag.ts          — Типы для тегов
│   ├── group.ts        — Типы для групп
│   ├── game.ts         — Типы для игр
│   ├── config.ts       — Типы конфигурации
│   ├── filter.ts       — Типы фильтров
│   ├── job.ts          — Типы задач
│   └── ...
├── hooks/
│   ├── useScenes.ts    — Хуки для сцен
│   ├── usePerformers.ts— Хуки для перформеров
│   ├── useGalleries.ts — Хуки для галерей
│   ├── useImages.ts    — Хуки для изображений
│   ├── useStudios.ts   — Хуки для студий
│   ├── useTags.ts      — Хуки для тегов
│   ├── useGroups.ts    — Хуки для групп
│   ├── useGames.ts     — Хуки для игр
│   ├── useConfig.ts    — Хуки для конфигурации
│   ├── useMetadata.ts  — Хуки для метаданных
│   ├── useScrapers.ts  — Хуки для скраперов
│   ├── usePlugins.ts   — Хуки для плагинов
│   ├── useJobs.ts      — Хуки для задач
│   ├── useSSE.ts       — Хуки для SSE-событий
│   └── ...
└── index.ts            — Re-exports
```

### Маппинг старых хуков на новые

```typescript
// Было (Apollo):
import * as GQL from "src/core/generated-graphql";
const { data, loading } = GQL.useFindSceneQuery({ variables: { id } });
const scene = data?.findScene;

// Стало (TanStack Query):
import { useFindScene } from "src/core/api/hooks/useScenes";
const { data: scene, isLoading } = useFindScene(id);
```

```typescript
// Было (мутация через StashService):
import { useSceneUpdate } from "src/core/StashService";
const [updateScene] = useSceneUpdate();
await updateScene({ variables: { input: { id: "1", title: "new" } } });

// Стало:
import { useSceneUpdate } from "src/core/api/hooks/useScenes";
const { mutateAsync: updateScene } = useSceneUpdate();
await updateScene({ id: "1", title: "new" });
```

### Кеширование

TanStack Query использует query keys для кеширования:

```typescript
// Автоматическая инвалидация при мутации
export function useSceneUpdate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: SceneUpdateInput) => 
      api.put<Scene>(`/scenes/${input.id}`, input),
    onSuccess: (data, vars) => {
      // Обновить кеш конкретной сцены
      qc.setQueryData(sceneKeys.detail(vars.id), data);
      // Инвалидировать списки
      qc.invalidateQueries({ queryKey: sceneKeys.lists() });
    },
  });
}
```

---

## 9. Миграция плагинов

### Обратная совместимость

**Важно:** Плагины — это экосистема. Нужен переходный период.

**Стратегия:**
1. Добавить REST API для плагинов (`api` объект)
2. Оставить `gql` объект для JS-плагинов как deprecated-обёртку поверх REST
3. Обновить примеры плагинов
4. В будущей версии удалить `gql` объект

### JS-плагины: новый API

```javascript
// Новый API объект, инжектируемый в плагины
var tag = api.get("/tags/1");
var tags = api.post("/tags/query", { 
  filter: { q: "example" } 
});
var newTag = api.post("/tags", { name: "new tag" });
api.put("/tags/" + tag.id, { name: "updated" });
api.delete("/tags/" + tag.id);
```

### Python-плагины: обновление StashInterface

```python
class StashInterface:
    def __init__(self, conn):
        self.base_url = f"http://localhost:{conn['Port']}/api/v1"
        self.session = conn.get("SessionCookie", {})
    
    def _get(self, path):
        return requests.get(f"{self.base_url}{path}", cookies=self.session).json()
    
    def _post(self, path, data=None):
        return requests.post(f"{self.base_url}{path}", json=data, cookies=self.session).json()
    
    def findTagIdWithName(self, name):
        result = self._post("/tags/query", {
            "tag_filter": {"name": {"value": name, "modifier": "EQUALS"}}
        })
        tags = result.get("data", [])
        return tags[0]["id"] if tags else None
```

---

## 10. Что НЕ мигрируем

### StashBox клиент (`pkg/stashbox/`)

**Причина**: StashBox — это внешний сервис со своим GraphQL API. Мы не контролируем его API.

**Что сохраняем:**
- `pkg/stashbox/graphql/` — сгенерированный клиент
- `.gqlgenc.yml` — конфиг генерации клиента
- `graphql/stash-box/query.graphql` — запросы к StashBox

**Зависимости, которые остаются:**
- `github.com/Yamashou/gqlgenc` — только для StashBox клиента
- `github.com/vektah/gqlparser/v2` — транзитивная зависимость gqlgenc

### Stash Scraper (частично)

**Причина**: Stash scraper скрапит данные с ДРУГОГО Stash-сервера через его GraphQL API.

**Решение**: Оставить как есть, но добавить поддержку REST API для новых серверов (через определение версии API удалённого сервера).

---

## 11. Риски и стратегия отката

### Риски

| Риск | Вероятность | Влияние | Митигация |
|---|---|---|---|
| Поломка плагинов | Высокая | Среднее | Deprecated-обёртка gql → REST |
| Потеря кеширования Apollo | Средняя | Среднее | TanStack Query query keys |
| Регрессия в фильтрации | Средняя | Высокое | Полная совместимость формата фильтров |
| Потеря оптимизации DataLoaders | Низкая | Низкое | REST-эндпоинты загружают полные объекты |
| Увеличение трафика (REST vs GraphQL selection) | Низкая | Низкое | Локальное приложение, трафик не критичен |

### Стратегия отката

- Каждый этап — отдельная ветка/коммит
- GraphQL остаётся работающим до полной миграции всех эндпоинтов
- Откат = возврат к GraphQL-хукам в компонентах

### Тестирование

- Каждый REST-эндпоинт тестируется руками через UI
- Сравнение ответов REST и GraphQL для тех же запросов
- Полный прогон функциональности после каждого этапа

---

## Итого: что получим

### Удалённый код
- ~128,000 строк сгенерированного Go-кода
- ~18,500 строк сгенерированных TypeScript-типов
- ~4,200 строк StashService.ts
- ~60 файлов резолверов
- 68 .graphql файлов фронтенда
- 35 .graphql файлов схемы
- Все GraphQL-зависимости (Go + npm)

### Новый код
- REST-хэндлеры (значительно проще резолверов — нет автогенерации, прямой маппинг)
- SSE-сервер (один файл)
- HTTP-клиент (один файл)
- TanStack Query хуки (модульные, по сущностям)
- TypeScript типы (ручные, прозрачные)

### Преимущества
1. **Простота** — REST понятен всем, не нужно разбираться в GraphQL
2. **Производительность** — нет overhead от GraphQL-парсинга и execution
3. **Размер бандла** — удаление Apollo Client (~50KB gzipped)
4. **Отладка** — REST-запросы видны в DevTools как обычные HTTP
5. **Плагины** — REST API проще для разработчиков плагинов
6. **Нет кодогенерации** — убираем сложный pipeline gqlgen + graphql-codegen
