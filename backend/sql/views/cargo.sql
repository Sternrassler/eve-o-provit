-- EVE Cargo & Hauling System - SQL Views (Adapted for eve-sde schema)

-- v_item_volumes: Item volume data for transport calculations
CREATE VIEW IF NOT EXISTS v_item_volumes AS
SELECT 
    t._key as type_id,
    COALESCE(json_extract(t.name, '$.en'), json_extract(t.name, '$.de')) as item_name,
    t.volume,
    t.capacity,
    t.volume as packagedVolume,  -- Simplified: use volume as packaged
    t.basePrice,
    g.categoryID as category_id,
    COALESCE(json_extract(g.name, '$.en'), json_extract(g.name, '$.de')) as category_name,
    t.marketGroupID,
    CASE WHEN t.volume > 0 THEN CAST(t.basePrice AS REAL) / t.volume ELSE 0 END as isk_per_m3
FROM types t
LEFT JOIN groups g ON t.groupID = g._key
WHERE t.published = 1;

-- v_ship_cargo_capacities: Ship cargo capacity information
CREATE VIEW IF NOT EXISTS v_ship_cargo_capacities AS
SELECT
    t._key as ship_type_id,
    COALESCE(json_extract(t.name, '$.en'), json_extract(t.name, '$.de')) as ship_name,
    t.volume as ship_volume,
    t.capacity as base_cargo_capacity,
    g._key as group_id,
    COALESCE(json_extract(g.name, '$.en'), json_extract(g.name, '$.de')) as group_name,
    c._key as category_id
FROM types t
JOIN groups g ON t.groupID = g._key
JOIN categories c ON g.categoryID = c._key
WHERE c._key = 6  -- Ships category
AND t.published = 1;
