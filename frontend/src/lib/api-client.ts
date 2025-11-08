// API client utilities for backend communication
import { 
  CharacterLocation, 
  CharacterShip, 
  CharacterFittingResponse 
} from "@/types/character";
import { Region, Ship } from "@/types/trading";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

// Backend response types
interface BackendRegionsResponse {
  regions: Array<{ id: number; name: string }>;
  count: number;
}

interface BackendShipsResponse {
  ships: Array<{
    type_id: number;
    type_name: string;
    cargo_capacity: number;
  }>;
  count: number;
}

/**
 * Fetch all regions from SDE backend
 */
export async function fetchRegions(): Promise<Region[]> {
  const response = await fetch(`${API_BASE_URL}/api/v1/sde/regions`);
  
  if (!response.ok) {
    throw new Error(`Failed to fetch regions: ${response.statusText}`);
  }
  
  const data: BackendRegionsResponse = await response.json();
  return data.regions || [];
}

/**
 * Fetch character location (requires authentication)
 */
export async function fetchCharacterLocation(authHeader: string): Promise<CharacterLocation> {
  const response = await fetch(`${API_BASE_URL}/api/v1/character/location`, {
    headers: {
      Authorization: authHeader,
    },
  });
  
  if (!response.ok) {
    throw new Error(`Failed to fetch character location: ${response.statusText}`);
  }
  
  return response.json();
}

/**
 * Fetch character's current ship (requires authentication)
 */
export async function fetchCharacterShip(authHeader: string): Promise<CharacterShip> {
  const response = await fetch(`${API_BASE_URL}/api/v1/character/ship`, {
    headers: {
      Authorization: authHeader,
    },
  });
  
  if (!response.ok) {
    throw new Error(`Failed to fetch character ship: ${response.statusText}`);
  }
  
  return response.json();
}

/**
 * Fetch all character ships in hangars (requires authentication)
 */
export async function fetchCharacterShips(authHeader: string): Promise<Ship[]> {
  const response = await fetch(`${API_BASE_URL}/api/v1/character/ships`, {
    headers: {
      Authorization: authHeader,
    },
  });
  
  if (!response.ok) {
    throw new Error(`Failed to fetch character ships: ${response.statusText}`);
  }
  
  const data: BackendShipsResponse = await response.json();
  
  // Convert backend format to Ship format
  return data.ships?.map((ship) => ({
    type_id: ship.type_id,
    name: ship.type_name,
    cargo_capacity: ship.cargo_capacity,
  })) || [];
}

/**
 * Fetch character's ship fitting (requires authentication)
 * @param authHeader - Authorization header (Bearer token)
 * @param characterId - Character ID
 * @param shipTypeId - Ship type ID
 */
export async function fetchCharacterFitting(
  authHeader: string,
  characterId: number,
  shipTypeId: number
): Promise<CharacterFittingResponse> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/characters/${characterId}/fitting/${shipTypeId}`,
    {
      headers: {
        Authorization: authHeader,
      },
    }
  );
  
  if (!response.ok) {
    throw new Error(`Failed to fetch character fitting: ${response.statusText}`);
  }
  
  return response.json();
}
