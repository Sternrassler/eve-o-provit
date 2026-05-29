// API client utilities for backend communication
import {
  CharacterLocation,
  CharacterShip,
  CharacterFittingResponse
} from "@/types/character";
import {
  Region,
  Ship,
  ItemSearchResult,
  HubComparisonResult,
  PortfolioRequest,
  PortfolioResult,
} from "@/types/trading";

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
export async function fetchCharacterLocation(): Promise<CharacterLocation> {
  const response = await fetch(`${API_BASE_URL}/api/v1/character/location`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch character location: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Fetch character's current ship (requires authentication)
 */
export async function fetchCharacterShip(): Promise<CharacterShip> {
  const response = await fetch(`${API_BASE_URL}/api/v1/character/ship`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch character ship: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Fetch all character ships in hangars (requires authentication)
 */
export async function fetchCharacterShips(): Promise<Ship[]> {
  const response = await fetch(`${API_BASE_URL}/api/v1/character/ships`, {
    credentials: "include",
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
 * @param characterId - Character ID
 * @param shipTypeId - Ship type ID
 */
export async function fetchCharacterFitting(
  characterId: number,
  shipTypeId: number
): Promise<CharacterFittingResponse> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/characters/${characterId}/fitting/${shipTypeId}`,
    {
      credentials: "include",
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch character fitting: ${response.statusText}`);
  }

  return response.json();
}

interface BackendItemSearchResponse {
  items: ItemSearchResult[];
  count?: number;
}

/**
 * Search EVE items by name fragment (min 3 chars).
 * @param q - search query
 * @param limit - max results (default 20)
 */
export async function searchItems(
  q: string,
  limit = 20
): Promise<ItemSearchResult[]> {
  const params = new URLSearchParams({ q, limit: String(limit) });
  const response = await fetch(
    `${API_BASE_URL}/api/v1/items/search?${params.toString()}`
  );

  if (!response.ok) {
    throw new Error(`Failed to search items: ${response.statusText}`);
  }

  const data: BackendItemSearchResponse = await response.json();
  return data.items || [];
}

/**
 * Compare an item's station-trading profitability across the major hubs
 * (requires authentication). Hubs are pre-sorted by net margin descending.
 * @param typeId - EVE item type_id
 */
export async function compareHubs(
  typeId: number
): Promise<HubComparisonResult> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/trading/hubs/compare`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ type_id: typeId }),
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to compare hubs: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Optimize capital allocation across items for maximum daily profit
 * (requires authentication). Returns a portfolio with items pre-sorted by
 * efficiency (most efficient first) plus a diversification score.
 * @param req - portfolio optimization parameters
 */
export async function optimizePortfolio(
  req: PortfolioRequest
): Promise<PortfolioResult> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/trading/portfolio/optimize`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(req),
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to optimize portfolio: ${response.statusText}`);
  }

  return response.json();
}

export interface SetWaypointOptions {
  /** Clear all existing waypoints before adding this one. */
  clearOtherWaypoints?: boolean;
  /** Insert this waypoint at the start of the existing route. */
  addToBeginning?: boolean;
}

/**
 * Set an in-game autopilot waypoint to the given destination (station/system)
 * via ESI (requires authentication and the esi-ui.write_waypoint.v1 scope, plus
 * a running EVE client). Call once per stop to build a multi-stop route:
 * first with clearOtherWaypoints=true, then false for each subsequent stop.
 * @param destinationId - station or system id to route to
 * @param opts - waypoint placement options
 */
export async function setWaypoint(
  destinationId: number,
  opts: SetWaypointOptions = {}
): Promise<void> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/esi/ui/autopilot/waypoint`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        destination_id: destinationId,
        clear_other_waypoints: opts.clearOtherWaypoints ?? false,
        add_to_beginning: opts.addToBeginning ?? false,
      }),
    }
  );

  if (!response.ok) {
    if (response.status === 401 || response.status === 403) {
      throw new Error("Missing scope esi-ui.write_waypoint.v1");
    }
    if (response.status === 404) {
      throw new Error("EVE client not running");
    }
    throw new Error(`Failed to set waypoint: ${response.statusText}`);
  }
}
