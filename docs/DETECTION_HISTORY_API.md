# 检测历史 API 前端请求示例

## API 接口说明

### 1. 获取检测历史列表

**接口地址：** `GET /api/v1/normal/detection-history`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 | 示例 |
|------|------|------|------|------|
| page | int | 否 | 页码，默认1 | `page=1` |
| page_size | int | 否 | 每页数量，默认10，最大100 | `page_size=20` |
| mode | string | 否 | 检测模式过滤 | `mode=basic` |
| start_time | string | 否 | 开始时间（ISO 8601格式） | `start_time=2026-03-01T00:00:00Z` |
| end_time | string | 否 | 结束时间（ISO 8601格式） | `end_time=2026-03-31T23:59:59Z` |

**响应格式：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "histories": [
      {
        "id": 1,
        "tenant_id": "917d94dc-3b64-4bb0-9e0b-4e4b748f9349",
        "text": "待检测文本...",
        "mode": "basic",
        "is_offensive": true,
        "result_json": "{\"is_offensive\": true, \"matches\": [...]}",
        "duration": 150,
        "created_at": "2026-03-24T10:00:00Z",
        "updated_at": "2026-03-24T10:00:00Z",
        "matches": [
          {
            "id": 1,
            "history_id": 1,
            "type": "政治",
            "text": "敏感词",
            "confidence": 0.95,
            "created_at": "2026-03-24T10:00:00Z"
          }
        ]
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10,
    "total_pages": 10
  }
}
```

### 2. 获取单个检测历史详情

**接口地址：** `GET /api/v1/normal/detection-history/:id`

**路径参数：**
- `id`: 检测历史ID（数字）

**响应格式：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "id": 1,
    "tenant_id": "917d94dc-3b64-4bb0-9e0b-4e4b748f9349",
    "text": "待检测文本...",
    "mode": "basic",
    "is_offensive": true,
    "result_json": "{\"is_offensive\": true, \"matches\": [...]}",
    "duration": 150,
    "created_at": "2026-03-24T10:00:00Z",
    "updated_at": "2026-03-24T10:00:00Z",
    "matches": [
      {
        "id": 1,
        "history_id": 1,
        "type": "政治",
        "text": "敏感词",
        "confidence": 0.95,
        "created_at": "2026-03-24T10:00:00Z"
      }
    ]
  }
}
```

### 3. 删除检测历史

**接口地址：** `DELETE /api/v1/normal/detection-history/:id`

**路径参数：**
- `id`: 检测历史ID（数字）

**响应格式：**
```json
{
  "code": 0,
  "message": "Deleted successfully"
}
```

### 4. 清空检测历史

**接口地址：** `DELETE /api/v1/normal/detection-history`

**响应格式：**
```json
{
  "code": 0,
  "message": "Cleared successfully"
}
```

## 前端请求示例

### cURL 示例

#### 1. 获取检测历史列表（基础分页）
```bash
curl -X GET "http://localhost:13142/api/v1/normal/detection-history?page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 2. 获取检测历史列表（带模式过滤）
```bash
curl -X GET "http://localhost:13142/api/v1/normal/detection-history?page=1&page_size=10&mode=basic" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 3. 获取检测历史列表（带时间范围过滤）
```bash
curl -X GET "http://localhost:13142/api/v1/normal/detection-history?page=1&page_size=10&start_time=2026-03-01T00:00:00Z&end_time=2026-03-31T23:59:59Z" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 4. 获取单个检测历史详情
```bash
curl -X GET "http://localhost:13142/api/v1/normal/detection-history/1" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 5. 删除单个检测历史
```bash
curl -X DELETE "http://localhost:13142/api/v1/normal/detection-history/1" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 6. 清空检测历史
```bash
curl -X DELETE "http://localhost:13142/api/v1/normal/detection-history" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### JavaScript/TypeScript 示例

```typescript
interface DetectionHistory {
  id: number;
  tenant_id: string;
  text: string;
  mode: string;
  is_offensive: boolean;
  result_json: string;
  duration: number;
  created_at: string;
  updated_at: string;
  matches: DetectionMatch[];
}

interface DetectionMatch {
  id: number;
  history_id: number;
  type: string;
  text: string;
  confidence: number;
  created_at: string;
}

interface DetectionHistoryListResponse {
  code: number;
  message: string;
  data: {
    histories: DetectionHistory[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
  };
}

// 获取检测历史列表
async function getDetectionHistories(params: {
  page?: number;
  page_size?: number;
  mode?: string;
  start_time?: string;
  end_time?: string;
}): Promise<DetectionHistoryListResponse> {
  const queryParams = new URLSearchParams({
    page: String(params.page || 1),
    page_size: String(params.page_size || 10),
    ...(params.mode && { mode: params.mode }),
    ...(params.start_time && { start_time: params.start_time }),
    ...(params.end_time && { end_time: params.end_time }),
  });

  const response = await fetch(
    `http://localhost:13142/api/v1/normal/detection-history?${queryParams}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    }
  );

  return response.json();
}

// 获取单个检测历史详情
async function getDetectionHistoryById(id: number): Promise<DetectionHistory> {
  const response = await fetch(
    `http://localhost:13142/api/v1/normal/detection-history/${id}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    }
  );

  const result = await response.json();
  return result.data;
}

// 删除检测历史
async function deleteDetectionHistory(id: number): Promise<void> {
  const response = await fetch(
    `http://localhost:13142/api/v1/normal/detection-history/${id}`,
    {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    }
  );

  const result = await response.json();
  if (result.code !== 0) {
    throw new Error(result.message);
  }
}

// 清空检测历史
async function clearDetectionHistories(): Promise<void> {
  const response = await fetch(
    `http://localhost:13142/api/v1/normal/detection-history`,
    {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    }
  );

  const result = await response.json();
  if (result.code !== 0) {
    throw new Error(result.message);
  }
}

// 使用示例
const histories = await getDetectionHistories({
  page: 1,
  page_size: 20,
  mode: 'basic',
  start_time: '2026-03-01T00:00:00Z',
  end_time: '2026-03-31T23:59:59Z'
});

console.log(histories);
```

### React 示例

```jsx
import { useState, useEffect } from 'react';

function DetectionHistoryList() {
  const [histories, setHistories] = useState([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState({
    mode: '',
    start_time: '',
    end_time: ''
  });

  const fetchHistories = async () => {
    setLoading(true);
    try {
      const queryParams = new URLSearchParams({
        page: String(page),
        page_size: '10',
        ...(filters.mode && { mode: filters.mode }),
        ...(filters.start_time && { start_time: filters.start_time }),
        ...(filters.end_time && { end_time: filters.end_time }),
      });

      const response = await fetch(
        `http://localhost:13142/api/v1/normal/detection-history?${queryParams}`,
        {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        }
      );

      const result = await response.json();
      if (result.code === 0) {
        setHistories(result.data.histories);
        setTotal(result.data.total);
      } else {
        alert(result.message);
      }
    } catch (error) {
      console.error('Error:', error);
      alert('网络错误');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHistories();
  }, [page, filters]);

  const handleDelete = async (id) => {
    if (!confirm('确定要删除这条检测历史吗？')) return;

    try {
      const response = await fetch(
        `http://localhost:13142/api/v1/normal/detection-history/${id}`,
        {
          method: 'DELETE',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        }
      );

      const result = await response.json();
      if (result.code === 0) {
        alert('删除成功');
        fetchHistories();
      } else {
        alert(result.message);
      }
    } catch (error) {
      console.error('Error:', error);
      alert('网络错误');
    }
  };

  const handleClearAll = async () => {
    if (!confirm('确定要清空所有检测历史吗？此操作不可恢复！')) return;

    try {
      const response = await fetch(
        `http://localhost:13142/api/v1/normal/detection-history`,
        {
          method: 'DELETE',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        }
      );

      const result = await response.json();
      if (result.code === 0) {
        alert('清空成功');
        fetchHistories();
      } else {
        alert(result.message);
      }
    } catch (error) {
      console.error('Error:', error);
      alert('网络错误');
    }
  };

  return (
    <div>
      <h2>检测历史</h2>
      
      {/* 过滤器 */}
      <div style={{ marginBottom: '20px' }}>
        <input
          type="text"
          placeholder="检测模式"
          value={filters.mode}
          onChange={(e) => setFilters({ ...filters, mode: e.target.value })}
        />
        <input
          type="datetime-local"
          placeholder="开始时间"
          value={filters.start_time}
          onChange={(e) => setFilters({ ...filters, start_time: e.target.value })}
        />
        <input
          type="datetime-local"
          placeholder="结束时间"
          value={filters.end_time}
          onChange={(e) => setFilters({ ...filters, end_time: e.target.value })}
        />
        <button onClick={() => setPage(1)}>搜索</button>
        <button onClick={handleClearAll}>清空所有</button>
      </div>

      {/* 列表 */}
      {loading ? (
        <div>加载中...</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>文本</th>
              <th>模式</th>
              <th>是否违规</th>
              <th>耗时(ms)</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {histories.map((history) => (
              <tr key={history.id}>
                <td>{history.id}</td>
                <td>{history.text.substring(0, 50)}...</td>
                <td>{history.mode}</td>
                <td>{history.is_offensive ? '是' : '否'}</td>
                <td>{history.duration}</td>
                <td>{new Date(history.created_at).toLocaleString()}</td>
                <td>
                  <button onClick={() => handleDelete(history.id)}>删除</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {/* 分页 */}
      <div style={{ marginTop: '20px' }}>
        <button 
          onClick={() => setPage(page - 1)} 
          disabled={page === 1}
        >
          上一页
        </button>
        <span>第 {page} 页，共 {Math.ceil(total / 10)} 页</span>
        <button 
          onClick={() => setPage(page + 1)} 
          disabled={page >= Math.ceil(total / 10)}
        >
          下一页
        </button>
      </div>
    </div>
  );
}

export default DetectionHistoryList;
```

### Vue.js 示例

```vue
<template>
  <div>
    <h2>检测历史</h2>
    
    <!-- 过滤器 -->
    <div class="filters">
      <input 
        v-model="filters.mode" 
        placeholder="检测模式"
        @keyup.enter="fetchHistories"
      />
      <input 
        v-model="filters.start_time" 
        type="datetime-local"
        @change="fetchHistories"
      />
      <input 
        v-model="filters.end_time" 
        type="datetime-local"
        @change="fetchHistories"
      />
      <button @click="fetchHistories">搜索</button>
      <button @click="handleClearAll">清空所有</button>
    </div>

    <!-- 列表 -->
    <div v-if="loading" class="loading">加载中...</div>
    <table v-else class="history-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>文本</th>
          <th>模式</th>
          <th>是否违规</th>
          <th>耗时(ms)</th>
          <th>创建时间</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="history in histories" :key="history.id">
          <td>{{ history.id }}</td>
          <td>{{ history.text.substring(0, 50) }}...</td>
          <td>{{ history.mode }}</td>
          <td>{{ history.is_offensive ? '是' : '否' }}</td>
          <td>{{ history.duration }}</td>
          <td>{{ formatDate(history.created_at) }}</td>
          <td>
            <button @click="handleDelete(history.id)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- 分页 -->
    <div class="pagination">
      <button 
        @click="page--" 
        :disabled="page === 1"
      >
        上一页
      </button>
      <span>第 {{ page }} 页，共 {{ totalPages }} 页</span>
      <button 
        @click="page++" 
        :disabled="page >= totalPages"
      >
        下一页
      </button>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      histories: [],
      loading: false,
      page: 1,
      total: 0,
      filters: {
        mode: '',
        start_time: '',
        end_time: ''
      }
    };
  },
  computed: {
    totalPages() {
      return Math.ceil(this.total / 10);
    }
  },
  mounted() {
    this.fetchHistories();
  },
  methods: {
    async fetchHistories() {
      this.loading = true;
      try {
        const queryParams = new URLSearchParams({
          page: String(this.page),
          page_size: '10',
          ...(this.filters.mode && { mode: this.filters.mode }),
          ...(this.filters.start_time && { start_time: this.filters.start_time }),
          ...(this.filters.end_time && { end_time: this.filters.end_time }),
        });

        const response = await fetch(
          `http://localhost:13142/api/v1/normal/detection-history?${queryParams}`,
          {
            headers: {
              'Authorization': `Bearer ${localStorage.getItem('token')}`,
              'Content-Type': 'application/json'
            }
          }
        );

        const result = await response.json();
        if (result.code === 0) {
          this.histories = result.data.histories;
          this.total = result.data.total;
        } else {
          this.$message.error(result.message);
        }
      } catch (error) {
        console.error('Error:', error);
        this.$message.error('网络错误');
      } finally {
        this.loading = false;
      }
    },
    async handleDelete(id) {
      if (!confirm('确定要删除这条检测历史吗？')) return;

      try {
        const response = await fetch(
          `http://localhost:13142/api/v1/normal/detection-history/${id}`,
          {
            method: 'DELETE',
            headers: {
              'Authorization': `Bearer ${localStorage.getItem('token')}`,
              'Content-Type': 'application/json'
            }
          }
        );

        const result = await response.json();
        if (result.code === 0) {
          this.$message.success('删除成功');
          this.fetchHistories();
        } else {
          this.$message.error(result.message);
        }
      } catch (error) {
        console.error('Error:', error);
        this.$message.error('网络错误');
      }
    },
    async handleClearAll() {
      if (!confirm('确定要清空所有检测历史吗？此操作不可恢复！')) return;

      try {
        const response = await fetch(
          `http://localhost:13142/api/v1/normal/detection-history`,
          {
            method: 'DELETE',
            headers: {
              'Authorization': `Bearer ${localStorage.getItem('token')}`,
              'Content-Type': 'application/json'
            }
          }
        );

        const result = await response.json();
        if (result.code === 0) {
          this.$message.success('清空成功');
          this.fetchHistories();
        } else {
          this.$message.error(result.message);
        }
      } catch (error) {
        console.error('Error:', error);
        this.$message.error('网络错误');
      }
    },
    formatDate(dateString) {
      return new Date(dateString).toLocaleString();
    }
  }
};
</script>

<style scoped>
.filters {
  margin-bottom: 20px;
}

.filters input {
  margin-right: 10px;
  padding: 5px;
}

.history-table {
  width: 100%;
  border-collapse: collapse;
}

.history-table th,
.history-table td {
  border: 1px solid #ddd;
  padding: 8px;
  text-align: left;
}

.history-table th {
  background-color: #f2f2f2;
}

.pagination {
  margin-top: 20px;
}

.pagination button {
  margin: 0 10px;
}

.loading {
  text-align: center;
  padding: 20px;
}
</style>
```

## 错误处理

### 错误码说明

| 错误码 | 说明 |
|--------|------|
| 1001 | 参数错误 |
| 1002 | 未授权 |
| 5000 | 检测历史不存在 |
| 5001 | 创建检测历史失败 |
| 5002 | 删除检测历史失败 |

### 错误响应示例

```json
{
  "code": 5000,
  "message": "检测历史不存在"
}
```

## 注意事项

1. **时间格式**：时间参数使用 ISO 8601 格式，例如 `2026-03-24T10:00:00Z`
2. **分页限制**：`page_size` 最大值为 100
3. **权限要求**：所有接口都需要 JWT token 认证
4. **软删除**：删除操作使用软删除，数据不会立即从数据库中删除
5. **级联删除**：删除检测历史时会级联删除关联的匹配记录
