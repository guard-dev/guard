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
    <div className="w-full">
      <div className="flex items-center justify-between py-4">
        <Input
          placeholder="Search services or regions..."
          value={globalFilter}
          onChange={(event) => setGlobalFilter(event.target.value)}
          className="max-w-sm"
        />
        {!scanCompleted && (
          <div className="flex items-center text-muted-foreground">
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Scan in progress...
          </div>
        )}
        {scanCompleted && (
          <div className="flex items-center text-muted-foreground">
            <Check className="mr-2 h-4 w-4" />
            Scan completed
          </div>
        )}
      </div>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  className="cursor-pointer hover:bg-muted/50"
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} onClick={() => router.push(`${pathname}/details/${row.original.scanItemId}`)}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={scanColumns.length}
                  className="h-24 text-center"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-end space-x-2 py-4">
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
        >
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.nextPage()}
          disabled={!table.getCanNextPage()}
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
