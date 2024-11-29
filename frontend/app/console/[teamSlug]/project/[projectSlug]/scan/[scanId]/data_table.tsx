"use client"

import * as React from "react"
import {
  CaretSortIcon,
} from "@radix-ui/react-icons"

import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table"

import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { SCAN_ITEMS } from "./page"
import { Input } from "@/components/ui/input"
import { Check, Loader2 } from "lucide-react"
import { usePathname, useRouter } from "next/navigation"

const scanColumns: ColumnDef<SCAN_ITEMS>[] = [
  {
    accessorKey: "service",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        >
          Service
          <CaretSortIcon className="ml-2 h-4 w-4" />
        </Button>
      )
    },
    enableGlobalFilter: true, // Add this line
  },
  {
    accessorKey: "region",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        >
          Region
          <CaretSortIcon className="ml-2 h-4 w-4" />
        </Button>
      )
    },
    enableGlobalFilter: true, // Add this line
  },
  {
    accessorKey: "summary",
    header: "Summary",
    cell: ({ row }) => (
      <div className="max-w-[300px] truncate" title={row.getValue("summary")}>
        {row.getValue("summary")}
      </div>
    ),
  },
  {
    accessorKey: "findings",
    header: "Findings",
    cell: ({ row }) => (row.getValue("findings") as string[]).length,
  },
  {
    accessorKey: "resourceCost",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        >
          Resource Cost
          <CaretSortIcon className="ml-2 h-4 w-4" />
        </Button>
      )
    },
    cell: ({ row }) => {
      const cost = row.getValue("resourceCost") as number;
      return cost ? Math.round(cost) : '-';
    },
  },
]

interface DataTableProps {
  data: SCAN_ITEMS[]
  scanCompleted: boolean;
}

export function DataTable({ data: _data, scanCompleted }: DataTableProps) {
  const [sorting, setSorting] = React.useState<SortingState>([])
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([])
  const [globalFilter, setGlobalFilter] = React.useState('')
  const pathname = usePathname()
  const router = useRouter();

  const data = React.useMemo(() => _data.map(cleanData), [_data]);

  const table = useReactTable({
    data,
    columns: scanColumns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    onSortingChange: setSorting,
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn: (row, columnId, filterValue) => {
      const value = row.getValue(columnId) as string
      return value.toLowerCase().includes(filterValue.toLowerCase())
    },
    state: {
      sorting,
      columnFilters,
      globalFilter,
    },
  })

  return (
    <div className="w-full space-y-4">
      <div className="flex items-center justify-between">
        <Input
          placeholder="Filter scans..."
          value={(table.getColumn("service")?.getFilterValue() as string) ?? ""}
          onChange={(event) =>
            table.getColumn("service")?.setFilterValue(event.target.value)
          }
          className="max-w-sm"
        />
        <div className="flex items-center gap-2">
          {!scanCompleted && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Scan in progress...
            </div>
          )}
          {scanCompleted && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Check className="h-4 w-4 text-green-500" />
              Scan completed
            </div>
          )}
        </div>
      </div>

      <div className="rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent border-neutral-200 dark:border-neutral-800">
              {table.getHeaderGroups().map((headerGroup) => (
                headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id} className="font-[family-name:var(--font-geist-mono)]">
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                    </TableHead>
                  )
                })
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && "selected"}
                  className="hover:bg-neutral-200/50 dark:hover:bg-neutral-800/50 cursor-pointer border-neutral-200 dark:border-neutral-800"
                  onClick={() => router.push(`${pathname}/details/${row.original.scanItemId}`)}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="font-[family-name:var(--font-geist-mono)]">
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={scanColumns.length}
                  className="h-24 text-center font-[family-name:var(--font-geist-mono)]"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-end space-x-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
          className="rounded-full font-[family-name:var(--font-geist-mono)]"
        >
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.nextPage()}
          disabled={!table.getCanNextPage()}
          className="rounded-full font-[family-name:var(--font-geist-mono)]"
        >
          Next
        </Button>
      </div>
    </div>
  )
}

const cleanData = (data: SCAN_ITEMS) => ({
  ...data,
  resourceCost: data.resourceCost,
  region: cleanString(data.region),
  remedy: cleanString(data.remedy),
  service: cleanString(data.service),
  summary: cleanString(data.summary),
  findings: data.findings.map((f: string) => cleanString(f)),
  scanItemEntries: data.scanItemEntries.map((entry: any) => ({
    ...entry,
    findings: entry.findings.map((f: string) => cleanString(f)),
    title: cleanString(entry.title),
    summary: cleanString(entry.summary),
    remedy: cleanString(entry.remedy),
    commands: entry.commands.map((c: string) => cleanString(c)).filter((c: string) => c.length > 0),
  })),
});

const cleanString = (string: string) => {
  // First, replace escaped quotes (\" or \') with unescaped quotes
  let cleanedString = string.replace(/\\"/g, '"').replace(/\\'/g, "'");

  // Now, remove the first and last quote if they match
  if ((cleanedString.startsWith('"') && cleanedString.endsWith('"')) ||
    (cleanedString.startsWith("'") && cleanedString.endsWith("'"))) {
    cleanedString = cleanedString.slice(1, -1);
  }

  return cleanedString;
};
