// types/character.ts
export interface CharacterLocation {
  character_id: number;
  solar_system_id: number;
  solar_system_name: string;
  region_id: number;
  region_name: string;
  station_id?: number;
  station_name?: string;
  structure_id?: number;
}

export interface CharacterShip {
  ship_type_id: number;
  ship_name: string;
  ship_item_id: number;
  ship_type_name: string;
  cargo_capacity: number;
}

/**
 * TradingSkills contains all trading-relevant character skills
 * All skill levels are 0-5 (0 = untrained, 5 = max level)
 */
export interface TradingSkills {
  // Trading Skills (Fees)
  Accounting: number;              // Sales Tax reduction (-10% per level, max -50%)
  BrokerRelations: number;         // Broker Fee reduction (-0.3% per level, max -1.5%)
  AdvancedBrokerRelations: number; // Additional Broker Fee reduction (-0.3% per level, max -1.5%)
  FactionStanding: number;         // Faction standing (-10.0 to +10.0, affects broker fees: -0.03% per 1.0)
  CorpStanding: number;            // Corp standing (-10.0 to +10.0, affects broker fees: -0.02% per 1.0)

  // Cargo Skills
  SpaceshipCommand: number;        // +5% cargo capacity per level (max +25%)
  CargoOptimization: number;       // Ship-specific cargo bonus (+5% per level, max +25%)

  // Navigation Skills
  Navigation: number;              // Warp speed increase (+5% per level, max +25%)
  EvasiveManeuvering: number;      // Align time reduction (-5% per level, max -25%)

  // Ship-specific Industrial Skills (each +5% cargo per level)
  GallenteIndustrial: number;      // Iteron, Nereus, etc.
  CaldariIndustrial: number;       // Badger, Crane, etc.
  AmarrIndustrial: number;         // Bestower, Sigil, etc.
  MinmatarIndustrial: number;      // Wreathe, Hoarder, etc.
}

/**
 * CharacterSkillsResponse from GET /api/v1/characters/:characterId/skills
 */
export interface CharacterSkillsResponse {
  character_id: number;
  skills: TradingSkills;
}

/**
 * FittedModule represents a single module fitted to a ship
 */
export interface FittedModule {
  type_id: number;
  type_name: string;
  slot: string; // HiSlot0-7, MedSlot0-7, LoSlot0-7, RigSlot0-2
  dogma_attributes: Record<number, number>; // attribute_id -> value
}

/**
 * FittingBonuses contains aggregated bonuses from all fitted modules
 */
export interface FittingBonuses {
  cargo_bonus_m3: number;         // Additive m³ bonus (e.g., 5000.0 from 2x Expanded Cargohold II)
  warp_speed_multiplier: number;  // Multiplicative warp speed (e.g., 1.488 = +48.8% from 3x Hyperspatial)
  inertia_modifier: number;       // Multiplicative inertia (e.g., 0.7566 = -24.34% from 2x Inertial Stabilizers)
}

/**
 * CharacterFittingResponse from GET /api/v1/characters/:characterId/fitting/:shipTypeId
 */
export interface CharacterFittingResponse {
  character_id: number;
  ship_type_id: number;
  effective_cargo_m3: number;     // Final effective cargo capacity (deterministic)
  warp_speed_au_s: number;        // Final warp speed in AU/s (deterministic)
  align_time_seconds: number;     // Final align time in seconds (deterministic)
  base_cargo_hold_m3: number;     // Base cargo capacity without bonuses
  base_warp_speed_au_s: number;   // Base warp speed without bonuses
  fitted_modules: FittedModule[];
  bonuses: FittingBonuses;
  cached: boolean;
}
