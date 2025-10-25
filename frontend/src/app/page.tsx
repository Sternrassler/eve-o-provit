import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col">
      {/* Hero Section */}
      <section className="flex flex-col items-center justify-center px-4 py-16 text-center md:py-24 lg:py-32">
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl lg:text-7xl">
          EVE-O-Provit
        </h1>
        <p className="mt-4 max-w-2xl text-lg text-muted-foreground sm:text-xl md:text-2xl">
          Market Analysis & Industry Calculator für EVE Online
        </p>
        <p className="mt-2 max-w-xl text-sm text-muted-foreground sm:text-base">
          Optimiere deine Trading-Strategien und Manufacturing-Prozesse mit Echtzeit-Marktdaten
        </p>
        <div className="mt-8 flex flex-col gap-4 sm:flex-row">
          <Button asChild size="lg">
            <Link href="/navigation">Navigation</Link>
          </Button>
          <Button asChild size="lg" variant="outline">
            <Link href="/cargo">Cargo Calculator</Link>
          </Button>
        </div>
      </section>

      {/* Features Section */}
      <section className="px-4 py-12 md:py-16 lg:py-20">
        <h2 className="mb-8 text-center text-2xl font-bold sm:text-3xl md:text-4xl">
          Features
        </h2>
        <div className="mx-auto grid max-w-6xl gap-6 sm:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle>Market Analysis</CardTitle>
              <CardDescription>Echtzeit ESI Daten</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Analysiere Marktpreise, Handelsvolumen und Preistrends für profitable Trades.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Navigation</CardTitle>
              <CardDescription>Route Planning</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Finde optimale Handelsrouten zwischen Systemen mit Trade-Hub Integration.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Industry Calculator</CardTitle>
              <CardDescription>Blueprint Calculation</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Berechne Material-Kosten und Profit für Manufacturing-Projekte.
              </p>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  );
}
