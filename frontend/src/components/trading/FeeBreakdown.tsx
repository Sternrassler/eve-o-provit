import { Separator } from "@/components/ui/separator";

interface FeeBreakdownProps {
  fees: {
    salesTax: number;
    brokerFees: number;
    estimatedRelistFee: number;
    totalFees: number;
  };
}

export function FeeBreakdown({ fees }: FeeBreakdownProps) {
  const formatISK = (value: number) => {
    return new Intl.NumberFormat("de-DE", {
      style: "decimal",
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value) + " ISK";
  };

  return (
    <div className="space-y-1 text-sm">
      <div className="flex justify-between">
        <span>Sales Tax:</span>
        <span>{formatISK(fees.salesTax)}</span>
      </div>
      <div className="flex justify-between">
        <span>Broker Fees:</span>
        <span>{formatISK(fees.brokerFees)}</span>
      </div>
      <div className="flex justify-between">
        <span>Est. Relist Fee:</span>
        <span>{formatISK(fees.estimatedRelistFee)}</span>
      </div>
      <Separator className="my-2" />
      <div className="flex justify-between font-bold">
        <span>Total:</span>
        <span>{formatISK(fees.totalFees)}</span>
      </div>
    </div>
  );
}
