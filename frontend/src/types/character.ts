// types/character.ts
export interface CharacterLocation {
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
