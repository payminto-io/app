import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { DataTable, stackRole, type DataTableColumn } from "./data-table";

type Row = { id: string; amount: string; status: string; ref: string; customer?: string; chain: string };

const columns: DataTableColumn<Row>[] = [
  { key: "amount", stack: "lead", header: "Amount", cell: (r) => r.amount },
  { key: "status", stack: "trail", header: "Status", cell: (r) => r.status },
  { key: "ref", stack: "meta", header: "Reference", cell: (r) => r.ref },
  { key: "customer", stack: "meta", header: "Customer", cell: (r) => r.customer ?? null },
  { key: "chain", header: "Chain", cell: (r) => r.chain },
];

const rows: Row[] = [
  { id: "a", amount: "10.00 USDC", status: "Paid", ref: "ref_a", customer: "a@example.com", chain: "Ethereum" },
  { id: "b", amount: "0.01 BTC", status: "Open", ref: "ref_b", chain: "Bitcoin" },
];

describe("DataTable stacked rows", () => {
  it("puts lead and trail on line one, meta on line two and the rest as labelled details", () => {
    const { container } = render(<DataTable columns={columns} rows={rows} getRowId={(r) => r.id} />);
    const stack = container.querySelector("[data-slot=data-table-stack]") as HTMLElement;
    const items = within(stack).getAllByRole("listitem");
    expect(items).toHaveLength(2);

    const [lineOne, lineTwo] = items[0].children;
    expect(lineOne.textContent).toBe("10.00 USDCPaid");
    expect(lineTwo.textContent).toBe("ref_aa@example.com");
    expect(within(items[0]).getByText("Chain").tagName).toBe("DT");

    expect(items[1].textContent).not.toContain("undefined");
    expect(within(items[1]).queryByText("a@example.com")).toBeNull();
  });

  it("keeps the full table for wide screens with a sticky first column", () => {
    render(<DataTable columns={columns} rows={rows} />);
    const firstHead = screen.getByRole("columnheader", { name: "Amount" });
    expect(firstHead.className).toContain("sticky");
    expect(screen.getByRole("columnheader", { name: "Status" }).className).not.toContain("sticky");
  });

  it("stacks unannotated columns sensibly", () => {
    const plain: DataTableColumn<Row>[] = [
      { key: "name", header: "Name", cell: () => "" },
      { key: "status", header: "Status", cell: () => "" },
      { key: "other", header: "Other", cell: () => "" },
      { key: "actions", header: "Actions", cell: () => "" },
    ];
    expect(plain.map(stackRole)).toEqual(["lead", "trail", "detail", "action"]);
  });
});
