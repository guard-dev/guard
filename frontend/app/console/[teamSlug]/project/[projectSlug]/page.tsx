'use client';

import { GetProjectInfoQuery, StartScanMutation, StartScanMutationVariables } from "@/gql/graphql";
import { useMutation, useQuery } from "@apollo/client";
import { GET_PROJECT_INFO, START_SCAN } from "./gql";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useEffect, useState } from "react";
import { Dialog, DialogTrigger, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ComboboxDemo } from "./combobox";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";
import { Skeleton } from "@/components/ui/skeleton";
import { FaAws } from "react-icons/fa";
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Check, Loader2, Trash2Icon } from "lucide-react";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { CaretSortIcon } from "@radix-ui/react-icons"
import { formatTimestamp, getTimeDifference } from "@/lib/utils";
import { ShowSubscriptionInfo } from "./current_plan";


const TeamPage = ({ params }: { params: { teamSlug: string; projectSlug: string } }) => {

  const teamSlug = params.teamSlug;
  const projectSlug = params.projectSlug;

  const { data, loading: dataLoading } = useQuery<GetProjectInfoQuery>(GET_PROJECT_INFO, { variables: { teamSlug, projectSlug }, pollInterval: 10000 });

  const currentSubscription = data?.teams[0].subscriptionPlans[0];

  const project = data?.teams[0].projects[0];
  const scans = project?.scans;
  const connections = project?.accountConnections;

  const [startScanMutation, { data: scanData, loading }] = useMutation<StartScanMutation>(START_SCAN);

  const handleStartScan = async () => {
    const variables: StartScanMutationVariables = {
      teamSlug,
      projectSlug: projectSlug || "",
      regions,
      services
    }
    await startScanMutation({ variables })
  };

  const subActive = currentSubscription !== undefined && currentSubscription?.stripeSubscriptionId !== null;

  const router = useRouter();

  const [regions, setRegions] = useState<string[]>([]);
  const [services, setServices] = useState<string[]>([]);

  const [sorting, setSorting] = useState<SortingState>([
    { id: "created", desc: true }
  ])

  const columns: ColumnDef<any>[] = [
    {
      accessorKey: "scanName",
      header: "Scan",
      cell: ({ row }) => formatTimestamp(row.original.created * 1000) + " scan",
    },
    {
      accessorKey: "serviceCount",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          >
            Services
            <CaretSortIcon className="ml-2 h-4 w-4" />
          </Button>
        )
      },
    },
    {
      accessorKey: "regionCount",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          >
            Regions
            <CaretSortIcon className="ml-2 h-4 w-4" />
          </Button>
        )
      },
    },
    {
      accessorKey: "scanStatus",
      header: "Status",
      cell: ({ row }) => (
        <div className="flex items-center">
          {!row.original.scanCompleted ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              In progress
            </>
          ) : (
            <>
              <Check className="mr-2 h-4 w-4" />
              Completed
            </>
          )}
        </div>
      ),
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
        const cost = row.original.resourceCost;
        return cost ? Math.round(cost) : '-';
      },
    },
    {
      accessorKey: "created",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          >
            Date
            <CaretSortIcon className="ml-2 h-4 w-4" />
          </Button>
        )
      },
      cell: ({ row }) => getTimeDifference(row.original.created * 1000),
    },
  ]

  const table = useReactTable({
    data: scans || [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    onSortingChange: setSorting,
    getSortedRowModel: getSortedRowModel(),
    state: {
      sorting,
    },
  })

  useEffect(() => {
    if (scanData) {
      router.push(`/console/${teamSlug}/project/${projectSlug}/scan/${scanData.startScan}`)
    }
  }, [scanData])

  return (
    <div className="flex w-full h-full items-center justify-center flex-col py-5">
      <div className="flex w-full h-full max-w-screen-xl flex-col">
        <Tabs defaultValue="scans" className="px-5">
          <TabsList>
            <TabsTrigger value="scans" disabled={dataLoading}>Scans</TabsTrigger>
            <TabsTrigger value="settings" disabled={dataLoading}>Settings</TabsTrigger>
          </TabsList>
          <TabsContent value="scans" className="py-4">
            {dataLoading ? (
              <LoadingSkeleton />
            ) : (
              <div>
                <div className="flex justify-between items-center mb-4">
                  <Dialog>
                    <DialogTrigger asChild>
                      <Button>New Scan</Button>
                    </DialogTrigger>
                    <DialogContent>
                      {
                        !subActive && !process.env.NEXT_PUBLIC_SELF_HOSTING ?
                          <ShowSubscriptionInfo
                            plan={currentSubscription!}
                            teamSlug={teamSlug}
                          />
                          :
                          <>
                            <DialogHeader>
                              <DialogTitle>New Scan</DialogTitle>
                              <DialogDescription>
                                Select the services and regions you want to scan.
                              </DialogDescription>
                            </DialogHeader>
                            <div>
                              Services
                            </div>
                            <div>
                              <ComboboxDemo availableOptions={availableServices} value={services} setValue={setServices} />
                            </div>
                            <div>
                              Regions
                            </div>
                            <div>
                              <ComboboxDemo availableOptions={availableRegions} value={regions} setValue={setRegions} />
                            </div>
                            <DialogFooter className="pt-5">
                              <DialogClose asChild>
                                <Button variant={"secondary"}>
                                  Cancel
                                </Button>
                              </DialogClose>
                              <Button onClick={handleStartScan} disabled={loading}>
                                Start Scan
                              </Button>
                            </DialogFooter>
                          </>
                      }
                    </DialogContent>
                  </Dialog>

                  <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    <div>
                      Resources Used: {currentSubscription?.subscriptionData?.resourcesUsed || 0}
                    </div>
                    <div>
                      Resources Included: {currentSubscription?.subscriptionData?.resourcesIncluded || 0}
                    </div>
                  </div>
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
                            onClick={() => {
                              router.push(`/console/${teamSlug}/project/${projectSlug}/scan/${row.original.scanId}`)
                            }}
                          >
                            {row.getVisibleCells().map((cell) => (
                              <TableCell key={cell.id}>
                                {flexRender(cell.column.columnDef.cell, cell.getContext())}
                              </TableCell>
                            ))}
                          </TableRow>
                        ))
                      ) : (
                        <TableRow>
                          <TableCell colSpan={columns.length} className="h-24 text-center">
                            No results.
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </div>
              </div>
            )}
          </TabsContent>
          <TabsContent value="settings" className="py-4">

            <div className="flex flex-col gap-2">
              <div className="text-xl">
                Connected Accounts
              </div>
              <Table>
                <TableCaption>A list of your connected cloud accounts.</TableCaption>
                <TableHeader>
                  <TableHead className="w-[100px]">Cloud</TableHead>
                  <TableHead>Account Number</TableHead>
                  <TableHead>External ID</TableHead>
                </TableHeader>
                <TableBody>
                  {connections?.map((c, idx) => (
                    <TableRow key={idx}>
                      <TableCell>
                        <FaAws size={30} />
                      </TableCell>
                      <TableCell>
                        {c.accountId}
                      </TableCell>
                      <TableCell>
                        {c.externalId}
                      </TableCell>
                      <TableCell>
                        <Button size={"icon"} variant={"ghost"} disabled>
                          <Trash2Icon />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {
              process.env.NEXT_PUBLIC_SELF_HOSTING ? <div /> :
                <div className="flex flex-col gap-2">
                  <div className="text-xl">
                    Current Plan
                  </div>
                  <ShowSubscriptionInfo
                    plan={currentSubscription!}
                    teamSlug={teamSlug}
                  />
                </div>
            }

          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

const availableRegions = [
  {
    value: "us-east-1",
    label: "us-east-1 (US East - N. Virginia)",
  },
  {
    value: "us-west-2",
    label: "us-west-2 (US West - Oregon)",
  },
  {
    value: "us-east-2",
    label: "us-east-2 (US East - Ohio)",
  },
  {
    value: "us-west-1",
    label: "us-west-1 (US West - N. California)",
  },
  {
    value: "ca-central-1",
    label: "ca-central-1 (Canada - Central)",
  },
  {
    value: "eu-west-1",
    label: "eu-west-1 (Europe - Ireland)",
  },
  {
    value: "eu-central-1",
    label: "eu-central-1 (Europe - Frankfurt)",
  },
  {
    value: "eu-west-2",
    label: "eu-west-2 (Europe - London)",
  },
  {
    value: "eu-north-1",
    label: "eu-north-1 (Europe - Stockholm)",
  },
  {
    value: "ap-northeast-1",
    label: "ap-northeast-1 (Asia Pacific - Tokyo)",
  },
  {
    value: "ap-southeast-1",
    label: "ap-southeast-1 (Asia Pacific - Singapore)",
  },
  {
    value: "ap-southeast-2",
    label: "ap-southeast-2 (Asia Pacific - Sydney)",
  },
  {
    value: "ap-south-1",
    label: "ap-south-1 (Asia Pacific - Mumbai)",
  },
];


const availableServices = [
  {
    value: "s3",
    label: "S3",
  },
  {
    value: "ec2",
    label: "EC2",
  },
  {
    value: "ecs",
    label: "ECS",
  },
  {
    value: "lambda",
    label: "Lambda",
  },
  {
    value: "dynamodb",
    label: "DynamoDB",
  },
  {
    value: "iam",
    label: "IAM",
  },
];

const LoadingSkeleton = () => (
  <div className="w-full">
    <div className="flex justify-start mb-4">
      <Skeleton className="h-10 w-[100px]" />
    </div>
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            {['Scan Name', 'Services', 'Regions', 'Scan Status', 'Scan Date'].map((header) => (
              <TableHead key={header}>
                <Skeleton className="h-8 w-full" />
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {[...Array(5)].map((_, index) => (
            <TableRow key={index}>
              {[...Array(5)].map((_, cellIndex) => (
                <TableCell key={cellIndex}>
                  <Skeleton className="h-6 w-full" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  </div>
);

export default TeamPage;
