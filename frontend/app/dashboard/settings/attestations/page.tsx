"use client";

import { ShieldCheck } from "lucide-react";
import { useAttestationStatus } from "@/lib/query/hooks/use-attestations";
import { explorerAddressUrl, explorerTxUrl, type AttestationStatus, type WorkflowStatus } from "@/lib/api/attestations";
import { ATTESTATIONS_COPY as C, intervalLabel, providerLabel } from "@/lib/copy/attestations";
import { PageHeader } from "@/components/page-header";
import { DetailItem, DetailList } from "@/components/detail-list";
import { SettingsSection } from "@/components/settings-section";
import { Notice } from "@/components/notice";
import { CopyField } from "@/components/copy-field";
import { DateTime } from "@/components/date-time";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { StatusBadge } from "@/components/ui/status-badge";
import { ErrorState, LoadingRows } from "@/components/ui/states";
import { SettingsTabs } from "../_components/settings-tabs";

export default function AttestationsSettingsPage() {
  const { data, error, isPending, refetch } = useAttestationStatus();

  return (
    <div className="space-y-5">
      <PageHeader title="Settings" />
      <SettingsTabs />
      <div className="space-y-8 pt-3">
        {error ? (
          <ErrorState message={C.error} retry={refetch} />
        ) : isPending ? (
          <LoadingRows rows={4} />
        ) : data === null ? (
          <OffState />
        ) : (
          <OnState status={data} />
        )}
      </div>
    </div>
  );
}

function OffState() {
  return (
    <SettingsSection title={C.title} description={C.explain}>
      <div className="flex items-start gap-3">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-sm border border-line bg-surface-sunken text-ink-soft">
          <ShieldCheck className="size-4" strokeWidth={1.75} aria-hidden />
        </div>
        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-body font-medium text-ink">{C.off.heading}</h3>
            <StatusBadge status="off" />
          </div>
          <p className="text-body-sm text-ink-soft">{C.off.body}</p>
          <p className="text-body-sm text-ink-soft">{C.off.howTo}</p>
        </div>
      </div>
    </SettingsSection>
  );
}

// Returns null for an empty address so DetailItem omits the row instead of showing a blank.
function addressField(chain: string, value: string) {
  if (!value) return null;
  const href = explorerAddressUrl(chain, value);
  return (
    <div className="space-y-1">
      <CopyField value={value} boxed={false} />
      {href ? (
        <a href={href} target="_blank" rel="noreferrer" className="tap text-body-sm text-tide hover:text-tide-strong">
          {C.fields.explorer}
        </a>
      ) : null}
    </div>
  );
}

function OnState({ status }: { status: AttestationStatus }) {
  const f = C.fields;
  return (
    <>
      <SettingsSection title={C.title} description={C.explain}>
        <div className="space-y-4">
          {status.degraded_from ? <Notice tone="wait">{C.provider.degraded(status.missing_keys)}</Notice> : null}
          {status.provider === "mock" ? <Notice tone="note">{C.provider.mockNote}</Notice> : null}
          <DetailList columns={2}>
            <DetailItem label={C.provider.label}>{providerLabel(status.provider)}</DetailItem>
            <DetailItem label={f.health}>
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge status={status.health.status} />
                {status.health.message ? <span className="text-body-sm text-ink-soft">{status.health.message}</span> : null}
              </div>
            </DetailItem>
            <DetailItem label={f.environment}>
              {status.environment ? <span className="capitalize">{status.environment}</span> : null}
            </DetailItem>
            <DetailItem label={f.chain}>{status.chain}</DetailItem>
            <DetailItem label={f.consumer}>{addressField(status.chain, status.consumer_address)}</DetailItem>
            <DetailItem label={f.forwarder}>{addressField(status.chain, status.forwarder_address)}</DetailItem>
            <DetailItem label={f.owner}>{addressField(status.chain, status.workflow_owner)}</DetailItem>
            <DetailItem label={f.signer}>
              {status.trigger_signer ? <code className="font-mono text-body-sm text-ink">{status.trigger_signer}</code> : null}
            </DetailItem>
            <DetailItem label={f.signerAddress}>{addressField(status.chain, status.trigger_signer_address)}</DetailItem>
            <DetailItem label={f.gatewayId}>
              {status.gateway_id ? <CopyField value={status.gateway_id} boxed={false} /> : null}
            </DetailItem>
            <DetailItem label={f.publicBase}>{status.public_base_url}</DetailItem>
            <DetailItem label={f.publicVerify}>{status.public_verify_enabled ? f.publicVerifyOn : f.publicVerifyOff}</DetailItem>
          </DetailList>
        </div>
      </SettingsSection>

      <SettingsSection title={C.workflows.heading} description={C.workflows.description}>
        <WorkflowsTable chain={status.chain} workflows={status.workflows} />
      </SettingsSection>
    </>
  );
}

function WorkflowsTable({ chain, workflows }: { chain: string; workflows: WorkflowStatus[] }) {
  const w = C.workflows;
  const columns: DataTableColumn<WorkflowStatus>[] = [
    {
      key: "workflow",
      stack: "lead",
      header: w.columns.workflow,
      cell: (r) => (
        <div className="flex flex-col">
          <span className="font-medium text-ink">{w.names[r.kind] ?? r.kind}</span>
          <span className="num text-caption text-ink-soft">{w.every(intervalLabel(r.interval_seconds))}</span>
        </div>
      ),
    },
    { key: "state", stack: "trail", header: w.columns.state, cell: (r) => <StatusBadge status={r.state} /> },
    {
      key: "credential",
      stack: "detail",
      header: w.columns.credential,
      className: "text-ink-soft",
      cell: (r) => (r.credential_configured ? w.credentialSet : w.credentialMissing),
    },
    {
      key: "run",
      stack: "detail",
      header: w.columns.lastRun,
      cell: (r) =>
        r.last_run ? (
          <div className="flex flex-col items-start gap-1">
            <DateTime value={r.last_run.started_at} />
            <StatusBadge status={r.last_run.status} />
          </div>
        ) : (
          <span className="text-ink-soft">{w.never}</span>
        ),
    },
    {
      key: "verified",
      stack: "detail",
      header: w.columns.lastVerified,
      cell: (r) => {
        const a = r.last_verified;
        const latest = r.last_attestation;
        // Only a verified record earns a date and a link; a newer unverified row is named by its status.
        const unverified = latest && latest.status !== "attested" && (!a || latest.id !== a.id) ? latest : null;
        if (!a && !unverified) return <span className="text-ink-soft">{w.noRecord}</span>;
        const href = a ? explorerTxUrl(chain, a.tx_hash) : null;
        return (
          <div className="flex flex-col items-start gap-1">
            {a ? <DateTime value={a.recorded_at} /> : <span className="text-ink-soft">{w.noRecord}</span>}
            <div className="flex flex-wrap items-center gap-2">
              {a ? <StatusBadge status={a.status} /> : null}
              {href ? (
                <a href={href} target="_blank" rel="noreferrer" className="tap text-body-sm text-tide hover:text-tide-strong">
                  {C.fields.explorer}
                </a>
              ) : null}
              {unverified ? <StatusBadge status={unverified.status} /> : null}
            </div>
          </div>
        );
      },
    },
  ];
  return <DataTable columns={columns} rows={workflows} getRowId={(r) => r.kind} footer={null} />;
}
