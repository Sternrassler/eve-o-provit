import { Separator } from "@/components/ui/separator";
import { formatISKWithSeparators } from "@/lib/utils";

interface FeeBreakdownProps {
  fees: {
    salesTax: number;
    brokerFees: number;
    estimatedRelistFee: number;
    totalFees: number;
  };
}

export function FeeBreakdown({ fees }: FeeBreakdownProps) {
  return (
    <div className="space-y-1 text-sm">
      <div className="flex justify-between">
        <span>Sales Tax:</span>
        <span>{formatISKWithSeparators(fees.salesTax)}</span>
      </div>
      <div className="flex justify-between">
        <span>Broker Fees:</span>
        <span>{formatISKWithSeparators(fees.brokerFees)}</span>
      </div>
      <div className="flex justify-between">
        <span>Est. Relist Fee:</span>
        <span>{formatISKWithSeparators(fees.estimatedRelistFee)}</span>
      </div>
      <Separator className="my-2" />
      <div className="flex justify-between font-bold">
        <span>Total:</span>
        <span>{formatISKWithSeparators(fees.totalFees)}</span>
      </div>
    </div>
  );
}
