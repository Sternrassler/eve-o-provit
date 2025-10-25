import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function CargoPage() {
  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold sm:text-4xl">Cargo Volume Calculator</h1>
      
      <Card>
        <CardHeader>
          <CardTitle>Coming Soon</CardTitle>
          <CardDescription>Cargo & Fitting Calculator</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            Berechne Cargo-Volumen für Ships und Fittings. SDE Integration folgt.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
