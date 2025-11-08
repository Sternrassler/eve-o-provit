import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { ShipFittingCard } from "./ShipFittingCard";
import * as apiClient from "@/lib/api-client";
import { CharacterFittingResponse } from "@/types/character";

vi.mock("@/lib/api-client");

const mockFittingResponse: CharacterFittingResponse = {
  character_id: 123456,
  ship_type_id: 648,
  fitted_modules: [
    {
      type_id: 1234,
      type_name: "Expanded Cargohold II",
      slot: "LoSlot0",
      dogma_attributes: {},
    },
    {
      type_id: 5678,
      type_name: "Hyperspatial Velocity Optimizer I",
      slot: "RigSlot0",
      dogma_attributes: {},
    },
  ],
  bonuses: {
    cargo_bonus_m3: 5000.0,
    warp_speed_multiplier: 1.488,
    inertia_modifier: 0.85,
  },
  cached: false,
};

describe("ShipFittingCard", () => {
  it("should render loading state initially", () => {
    vi.mocked(apiClient.fetchCharacterFitting).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    render(
      <ShipFittingCard
        characterId={123456}
        shipTypeId={648}
        authHeader="Bearer token"
      />
    );

    expect(screen.getByText("Schiff-Fitting")).toBeInTheDocument();
    expect(screen.getByText("Lade Fitting-Daten...")).toBeInTheDocument();
  });

  it("should render error state when fetch fails", async () => {
    vi.mocked(apiClient.fetchCharacterFitting).mockRejectedValue(
      new Error("Failed to fetch")
    );

    render(
      <ShipFittingCard
        characterId={123456}
        shipTypeId={648}
        authHeader="Bearer token"
      />
    );

    await waitFor(() => {
      expect(screen.getByText("Fehler beim Laden")).toBeInTheDocument();
      expect(screen.getByText(/Failed to fetch/)).toBeInTheDocument();
    });
  });

  it("should render empty state when no modules fitted", async () => {
    vi.mocked(apiClient.fetchCharacterFitting).mockResolvedValue({
      ...mockFittingResponse,
      fitted_modules: [],
    });

    render(
      <ShipFittingCard
        characterId={123456}
        shipTypeId={648}
        authHeader="Bearer token"
      />
    );

    await waitFor(() => {
      expect(screen.getByText("Keine Module gefittet")).toBeInTheDocument();
    });
  });

  it("should render fitting data successfully", async () => {
    vi.mocked(apiClient.fetchCharacterFitting).mockResolvedValue(
      mockFittingResponse
    );

    render(
      <ShipFittingCard
        characterId={123456}
        shipTypeId={648}
        authHeader="Bearer token"
      />
    );

    await waitFor(() => {
      // Check module display
      expect(screen.getByText("Expanded Cargohold II")).toBeInTheDocument();
      expect(screen.getByText("Hyperspatial Velocity Optimizer I")).toBeInTheDocument();

      // Check bonuses
      expect(screen.getByText("Cargo Bonus")).toBeInTheDocument();
      expect(screen.getByText("+5.000 m³")).toBeInTheDocument();

      expect(screen.getByText("Warp Speed")).toBeInTheDocument();
      expect(screen.getByText("+48,8 %")).toBeInTheDocument();

      expect(screen.getByText("Agility")).toBeInTheDocument();
      expect(screen.getByText("+15 %")).toBeInTheDocument();
    });
  });

  it("should not fetch when authHeader is null", () => {
    const fetchSpy = vi.mocked(apiClient.fetchCharacterFitting);

    render(
      <ShipFittingCard
        characterId={123456}
        shipTypeId={648}
        authHeader={null}
      />
    );

    expect(fetchSpy).not.toHaveBeenCalled();
  });
});
