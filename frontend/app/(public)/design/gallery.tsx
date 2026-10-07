"use client";

import { AppShell } from "@/components/layout/app-shell";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatusBadge } from "@/components/ui/status-badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState, ErrorState, LoadingRows } from "@/components/ui/states";
import { Rail } from "@/components/rail";
import { CurrencyDisplay } from "@/components/currency-display";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

const STATUSES = ["created", "open", "confirming", "partially_filled", "filled", "over_filled", "refunded", "cancelled", "expired", "failed", "pending_approval", "processed", "final", "reorged"];

/** Sample values below are illustrative and labelled as such; nothing here reads the API. */
export function DesignGallery() {
  return (
    <AppShell>
      <PageHeader title="Design gallery" description="Primitives rendered inside the real shell. Development only.">
        <Button variant="outline">Secondary</Button>
        <Button>Primary</Button>
      </PageHeader>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Status vocabulary</CardTitle>
            <CardDescription>Every backend status maps to a label, a tone and a glyph.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {STATUSES.map((s) => (
              <StatusBadge key={s} status={s} />
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>The rail</CardTitle>
            <CardDescription>Received, final, settled. Sample states, not live data.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <Rail step={1} size="lg" />
            <Rail step={2} size="lg" />
            <Rail step={3} size="lg" />
            <Rail step={1} failed size="lg" />
            <div className="flex items-center gap-3 text-body-sm text-ink-soft">
              Micro rail in a table cell <Rail step={2} />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Money</CardTitle>
            <CardDescription>Sample amounts to show formatting; see DESIGN.md section 8.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CurrencyDisplay amount="1250.5" currency="USD" size="display" />
            <div className="flex flex-wrap gap-6">
              <CurrencyDisplay amount="0.000123" currency="USDC" size="lg" />
              <CurrencyDisplay amount="" minor="1250000000" currency="USDC" size="lg" />
              <CurrencyDisplay amount="12" currency="USD" signed />
              <CurrencyDisplay amount="-12" currency="USD" signed />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Form controls</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="g-email">Email</Label>
              <Input id="g-email" type="email" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="g-bad">With an error</Label>
              <Input id="g-bad" aria-invalid defaultValue="not-an-address" />
            </div>
            <div className="flex flex-wrap gap-2">
              <Button size="sm">Small</Button>
              <Button size="sm" variant="secondary">Secondary</Button>
              <Button size="sm" variant="ghost">Ghost</Button>
              <Button size="sm" variant="destructive">Destructive</Button>
              <Button size="sm" variant="link">Link</Button>
              <Button size="sm" disabled>Disabled</Button>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Table</CardTitle>
          </CardHeader>
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Reference</TableHead>
                  <TableHead>Rail</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell className="font-mono text-label">sample_row_1</TableCell>
                  <TableCell><Rail step={3} /></TableCell>
                  <TableCell><StatusBadge status="filled" /></TableCell>
                  <TableCell className="text-right"><CurrencyDisplay amount="40" currency="USDC" size="sm" /></TableCell>
                </TableRow>
                <TableRow>
                  <TableCell className="font-mono text-label">sample_row_2</TableCell>
                  <TableCell><Rail step={1} /></TableCell>
                  <TableCell><StatusBadge status="confirming" /></TableCell>
                  <TableCell className="text-right"><CurrencyDisplay amount="19.99" currency="USD" size="sm" /></TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <EmptyState title="No payments yet" description="Payments appear here as soon as a link is paid." action={<Button size="sm">Create payment link</Button>} />
        <div className="space-y-4">
          <ErrorState message="The API returned 503 while loading payments." retry={() => undefined} />
          <LoadingRows rows={3} />
        </div>
      </div>
    </AppShell>
  );
}
