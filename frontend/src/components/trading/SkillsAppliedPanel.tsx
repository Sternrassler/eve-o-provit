"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SkillsApplied } from "@/types/trading";
import { GraduationCap } from "lucide-react";

interface SkillsAppliedPanelProps {
  skills: SkillsApplied;
}

function formatPercent(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`;
}

/**
 * Panel summarising which trade-skill levels were applied to the margin
 * calculation and the resulting effective fee rates.
 */
export function SkillsAppliedPanel({ skills }: SkillsAppliedPanelProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="flex items-center gap-2 text-sm font-medium">
          <GraduationCap className="size-4 text-muted-foreground" aria-hidden="true" />
          Skills angewendet
        </CardTitle>
        <Badge variant={skills.applied ? "default" : "secondary"}>
          {skills.applied ? "Aktiv" : "Standard"}
        </Badge>
      </CardHeader>
      <CardContent>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <div>
            <dt className="text-muted-foreground">Accounting</dt>
            <dd className="font-medium">Level {skills.accounting}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Broker Relations</dt>
            <dd className="font-medium">Level {skills.broker_relations}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Sales Tax</dt>
            <dd className="font-medium">{formatPercent(skills.sales_tax_rate)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Broker Fee</dt>
            <dd className="font-medium">{formatPercent(skills.broker_fee_rate)}</dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  );
}
