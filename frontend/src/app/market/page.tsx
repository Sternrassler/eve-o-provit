import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function MarketPage() {
  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold sm:text-4xl">Market Analysis</h1>
      
      <Card>
        <CardHeader>
          <CardTitle>Coming Soon</CardTitle>
          <CardDescription>ESI Market Data Integration</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            Echtzeit-Marktdaten, Preistrends und Handelsvolumen. ESI API Integration in Arbeit.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
