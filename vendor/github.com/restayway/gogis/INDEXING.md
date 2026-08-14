# PostGIS Indexing Guide for GoGIS

This guide covers spatial indexing strategies for optimal performance when using GoGIS with PostGIS.

## Why Spatial Indexes Matter

Spatial queries can be extremely slow without proper indexing. A spatial index uses a bounding box approach to quickly eliminate geometries that can't possibly match your query, dramatically reducing the search space.

**Without spatial index**: O(n) - checks every row
**With spatial index**: O(log n) - uses spatial tree structure

## GiST Indexes

PostGIS uses GiST (Generalized Search Tree) indexes for spatial data:

```sql
CREATE INDEX idx_table_geom ON table_name USING GIST (geometry_column);
```

### Basic Index Creation

For each geometry type in GoGIS:

```sql
-- Point indexes
CREATE INDEX idx_locations_point ON locations USING GIST (point);

-- LineString indexes  
CREATE INDEX idx_routes_path ON routes USING GIST (path);

-- Polygon indexes
CREATE INDEX idx_regions_area ON regions USING GIST (area);

-- GeometryCollection indexes
CREATE INDEX idx_places_geometries ON places USING GIST (geometries);
```

### Index with Include Columns

Include frequently queried non-spatial columns:

```sql
CREATE INDEX idx_locations_point_with_name 
ON locations USING GIST (point) 
INCLUDE (name, type);
```

## Index Performance Analysis

### Check Index Usage

```sql
-- See if your queries are using spatial indexes
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM locations 
WHERE ST_DWithin(point, ST_Point(-73.9665, 40.7812), 0.01);
```

Look for:
- `Index Scan using idx_locations_point` (good)
- `Seq Scan on locations` (bad - no index used)

### Index Statistics

```sql
-- Check index size and usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) as size
FROM pg_stat_user_indexes 
WHERE indexname LIKE 'idx_%_gist%';
```

## Advanced Indexing Strategies

### 1. Compound Indexes

Combine spatial and non-spatial columns:

```sql
-- B-tree on category, GiST on geometry
CREATE INDEX idx_locations_category ON locations (category);
CREATE INDEX idx_locations_point ON locations USING GIST (point);

-- Query both efficiently
SELECT * FROM locations 
WHERE category = 'restaurant' 
  AND ST_DWithin(point, ST_Point(-73.9665, 40.7812), 0.01);
```

### 2. Partial Indexes

Index only relevant subsets:

```sql
-- Index only active locations
CREATE INDEX idx_active_locations_point 
ON locations USING GIST (point) 
WHERE status = 'active';

-- Index only large polygons (for complex area queries)
CREATE INDEX idx_large_regions_area 
ON regions USING GIST (area) 
WHERE ST_Area(area) > 0.001;
```

### 3. Clustered Tables

Physically order table data by spatial locality:

```sql
-- Cluster table by spatial index
CLUSTER locations USING idx_locations_point;

-- Enable auto-clustering for new data
ALTER TABLE locations SET (fillfactor = 90);
```

## Index Maintenance

### Vacuum and Analyze

Spatial indexes need regular maintenance:

```sql
-- Update index statistics
ANALYZE locations;

-- Rebuild index if needed
REINDEX INDEX idx_locations_point;

-- Full table maintenance
VACUUM ANALYZE locations;
```

### Monitor Index Bloat

```sql
-- Check for index bloat
SELECT 
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size,
    pg_size_pretty(pg_relation_size(tablename::regclass)) as table_size
FROM pg_stat_user_indexes 
WHERE indexname LIKE '%gist%'
ORDER BY pg_relation_size(indexrelid) DESC;
```

## Query Optimization Tips

### 1. Use ST_DWithin Instead of ST_Distance

```sql
-- Efficient - uses spatial index
SELECT * FROM locations 
WHERE ST_DWithin(point, ST_Point(-73.9665, 40.7812), 0.01);

-- Inefficient - can't use spatial index effectively
SELECT * FROM locations 
WHERE ST_Distance(point, ST_Point(-73.9665, 40.7812)) < 0.01;
```

### 2. Bounding Box Pre-filtering

Use the `&&` operator for bounding box intersection:

```sql
-- Very fast bounding box check, then exact geometry check
SELECT * FROM regions 
WHERE area && ST_Expand(ST_Point(-73.9665, 40.7812), 0.01)
  AND ST_Contains(area, ST_Point(-73.9665, 40.7812));
```

### 3. Order of Operations

Place spatial predicates first in complex queries:

```sql
-- Good - spatial filter first
SELECT * FROM locations 
WHERE ST_DWithin(point, ST_Point(-73.9665, 40.7812), 0.01)
  AND category = 'restaurant'
  AND rating > 4.0;
```

## GoGIS-Specific Indexing

### Model Tags for Indexes

Use GORM tags to ensure indexes are created:

```go
type Location struct {
    ID       uint        `gorm:"primaryKey"`
    Name     string      `gorm:"index"`
    Category string      `gorm:"index"`
    Point    gogis.Point `gorm:"type:geometry(Point,4326);index:,type:gist"`
}

type Route struct {
    ID     uint             `gorm:"primaryKey"`
    Name   string           `gorm:"index"`
    Path   gogis.LineString `gorm:"type:geometry(LineString,4326);index:,type:gist"`
    Active bool             `gorm:"index"`
}
```

### Migration with Indexes

```go
// In your migration
func CreateIndexes(db *gorm.DB) error {
    queries := []string{
        "CREATE INDEX IF NOT EXISTS idx_locations_point ON locations USING GIST (point)",
        "CREATE INDEX IF NOT EXISTS idx_routes_path ON routes USING GIST (path)",
        "CREATE INDEX IF NOT EXISTS idx_regions_area ON regions USING GIST (area)",
        // Compound indexes
        "CREATE INDEX IF NOT EXISTS idx_locations_category_point ON locations (category) INCLUDE (point)",
    }
    
    for _, query := range queries {
        if err := db.Exec(query).Error; err != nil {
            return err
        }
    }
    return nil
}
```

## Performance Benchmarks

### Test Spatial Query Performance

```go
// Benchmark your spatial queries
func BenchmarkSpatialQuery(b *testing.B) {
    // Setup test data...
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var locations []Location
        db.Where("ST_DWithin(point, ST_Point(?, ?), ?)", 
            -73.9665, 40.7812, 0.01).Find(&locations)
    }
}
```

### Expected Performance

With proper indexes on ~1M rows:
- Point containment query: < 10ms
- Distance-based ordering: < 50ms  
- Complex polygon intersection: < 100ms

Without indexes on the same data:
- Point containment query: > 1000ms
- Distance-based ordering: > 5000ms
- Complex polygon intersection: > 10000ms

## Troubleshooting

### Index Not Being Used

1. **Check if index exists**:
   ```sql
   \d+ table_name
   ```

2. **Check query plan**:
   ```sql
   EXPLAIN (ANALYZE, BUFFERS) SELECT ...;
   ```

3. **Update statistics**:
   ```sql
   ANALYZE table_name;
   ```

4. **Check PostGIS version**:
   ```sql
   SELECT PostGIS_Version();
   ```

### Slow Spatial Queries

1. **Ensure proper SRID usage** - mixing SRIDs prevents index usage
2. **Check for missing ST_Transform** in distance calculations
3. **Verify index is not bloated** - REINDEX if necessary
4. **Consider partial indexes** for large datasets with filtering

### Memory Issues

1. **Increase work_mem** for complex spatial operations:
   ```sql
   SET work_mem = '256MB';
   ```

2. **Use LIMIT** with ORDER BY for large result sets:
   ```sql
   SELECT * FROM locations 
   WHERE ST_DWithin(point, ST_Point(-73.9665, 40.7812), 0.1)
   ORDER BY ST_Distance(point, ST_Point(-73.9665, 40.7812))
   LIMIT 100;
   ```

## Configuration Recommendations

### PostgreSQL Settings

```ini
# postgresql.conf optimizations for spatial workloads
shared_buffers = 25% of RAM
work_mem = 256MB
maintenance_work_mem = 1GB
effective_cache_size = 75% of RAM
random_page_cost = 1.1  # For SSD storage
```

### PostGIS Settings

```sql
-- Increase precision for better index performance
ALTER TABLE locations ALTER COLUMN point TYPE geometry(Point, 4326) 
USING ST_SetSRID(point, 4326);
```

This indexing strategy will ensure your GoGIS spatial queries perform optimally at scale.