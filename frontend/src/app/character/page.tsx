"use client";

import { useAuth } from "@/lib/auth-context";
import { useEffect, useState, useCallback } from "react";
import Image from "next/image";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { CharacterLocation, CharacterShip } from "@/types/character";

interface CharacterDetails {
  character_id: number;
  character_name: string;
  scopes: string[];
  portrait_url: string;
}

export default function CharacterPage() {
  const { character, isAuthenticated, isLoading, getAuthHeader } = useAuth();
  const [details, setDetails] = useState<CharacterDetails | null>(null);
  const [location, setLocation] = useState<CharacterLocation | null>(null);
  const [ship, setShip] = useState<CharacterShip | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [locationError, setLocationError] = useState<string | null>(null);
  const [shipError, setShipError] = useState<string | null>(null);

  const fetchCharacterDetails = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const authHeader = getAuthHeader();
      if (!authHeader) {
        setError("No authentication token available");
        return;
      }

      const response = await fetch("http://localhost:9001/api/v1/character", {
        headers: {
          Authorization: authHeader,
        },
      });

      if (!response.ok) {
        throw new Error(`API request failed: ${response.statusText}`);
      }

      const data = await response.json();
      setDetails(data);
    } catch (err) {
      console.error("Failed to fetch character details:", err);
      setError(err instanceof Error ? err.message : "Failed to load character details");
    } finally {
      setLoading(false);
    }
  }, [getAuthHeader]);

  const fetchCharacterLocation = useCallback(async () => {
    setLocationError(null);

    try {
      const authHeader = getAuthHeader();
      if (!authHeader) {
        setLocationError("No authentication token available");
        return;
      }

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";
      const response = await fetch(`${apiUrl}/api/v1/character/location`, {
        headers: {
          Authorization: authHeader,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch location: ${response.statusText}`);
      }

      const data = await response.json();
      setLocation(data);
    } catch (err) {
      console.error("Failed to fetch character location:", err);
      setLocationError("Keine Daten verfügbar");
    }
  }, [getAuthHeader]);

  const fetchCharacterShip = useCallback(async () => {
    setShipError(null);

    try {
      const authHeader = getAuthHeader();
      if (!authHeader) {
        setShipError("No authentication token available");
        return;
      }

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";
      const response = await fetch(`${apiUrl}/api/v1/character/ship`, {
        headers: {
          Authorization: authHeader,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch ship: ${response.statusText}`);
      }

      const data = await response.json();
      setShip(data);
    } catch (err) {
      console.error("Failed to fetch character ship:", err);
      setShipError("Keine Daten verfügbar");
    }
  }, [getAuthHeader]);

  useEffect(() => {
    if (isAuthenticated && character) {
      fetchCharacterDetails();
      fetchCharacterLocation();
      fetchCharacterShip();
    }
  }, [isAuthenticated, character, fetchCharacterDetails, fetchCharacterLocation, fetchCharacterShip]);

  if (isLoading) {
    return (
      <div className="container mx-auto p-8">
        <Card>
          <CardHeader>
            <Skeleton className="h-8 w-64" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-32 w-full" />
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <div className="container mx-auto p-8">
        <Card>
          <CardHeader>
            <CardTitle>Character Information</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground">Please login to view character details.</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-8 space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Character Information</CardTitle>
        </CardHeader>
        <CardContent>
          {character && (
            <div className="flex items-start gap-6">
              <Image
                src={character.portrait_url}
                alt={character.character_name}
                width={128}
                height={128}
                className="rounded-lg"
              />
              <div className="space-y-3 flex-1">
                <div>
                  <p className="text-sm text-muted-foreground">Name</p>
                  <p className="text-xl font-bold">{character.character_name}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Character ID</p>
                  <p className="font-mono">{character.character_id}</p>
                </div>
                
                {/* Clone Standort */}
                <div>
                  <p className="text-sm text-muted-foreground">Clone Standort</p>
                  {locationError ? (
                    <p className="text-sm text-muted-foreground">{locationError}</p>
                  ) : location ? (
                    <p className="font-medium">
                      {location.station_name || "Unbekannte Station"} ({location.solar_system_name})
                      {location.region_name && (
                        <span className="text-sm text-muted-foreground ml-2">
                          [{location.region_name}]
                        </span>
                      )}
                    </p>
                  ) : (
                    <Skeleton className="h-5 w-48" />
                  )}
                </div>

                {/* Aktuelles Schiff */}
                <div>
                  <p className="text-sm text-muted-foreground">Aktuelles Schiff</p>
                  {shipError ? (
                    <p className="text-sm text-muted-foreground">{shipError}</p>
                  ) : ship ? (
                    <p className="font-medium">
                      {ship.ship_name} ({ship.ship_type_name})
                    </p>
                  ) : (
                    <Skeleton className="h-5 w-48" />
                  )}
                </div>

                {character.scopes && character.scopes.length > 0 && (
                  <div>
                    <p className="text-sm text-muted-foreground">Scopes</p>
                    <div className="flex flex-wrap gap-2 mt-1">
                      {character.scopes.map((scope) => (
                        <span
                          key={scope}
                          className="px-2 py-1 text-xs bg-secondary rounded-md"
                        >
                          {scope}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>API Response</CardTitle>
            <Button onClick={fetchCharacterDetails} disabled={loading}>
              {loading ? "Loading..." : "Refresh"}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {error && (
            <div className="p-4 bg-destructive/10 text-destructive rounded-lg">
              <p className="font-semibold">Error</p>
              <p className="text-sm">{error}</p>
            </div>
          )}

          {loading && !details && (
            <Skeleton className="h-32 w-full" />
          )}

          {details && (
            <pre className="p-4 bg-muted rounded-lg overflow-x-auto">
              <code>{JSON.stringify(details, null, 2)}</code>
            </pre>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
