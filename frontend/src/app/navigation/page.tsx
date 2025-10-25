import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function NavigationPage() {
  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold sm:text-4xl">Navigation & Route Planning</h1>
      
      <Card>
        <CardHeader>
          <CardTitle>Coming Soon</CardTitle>
          <CardDescription>EVE Online Route Planner</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            Finde optimale Handelsrouten zwischen Systemen. Integration mit SDE Daten läuft.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
