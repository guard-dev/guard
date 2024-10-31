import { Badge } from "@/components/ui/badge";
import { gql, useMutation } from "@apollo/client";
import StripeCheckoutForm from "./stripe_checkout_form";
import { CreateStripeCheckoutMutation, GetProjectInfoQuery } from "@/gql/graphql";
import { titleCase } from "@/lib/utils";
import { loadStripe } from '@stripe/stripe-js';

const CREATE_STRIPE_CHECKOUT = gql`
  mutation CreateStripeCheckout($teamSlug: String!, $lookUpKey: String!) {
    createCheckoutSession(teamSlug: $teamSlug, lookUpKey: $lookUpKey) {
      sessionId
    }
  }
`;


type SUBSCRIPTION_PLANS =
  GetProjectInfoQuery['teams'][0]['subscriptionPlans'][0];

export const ShowSubscriptionInfo = ({
  plan,
  teamSlug,
}: {
  plan: SUBSCRIPTION_PLANS;
  teamSlug: string;
}) => {
  const subActive = plan !== undefined && plan?.stripeSubscriptionId !== null;
  const subscriptionData = plan?.subscriptionData;
  const subStatus = subscriptionData?.status;

  const [createCheckoutSession, { }] =
    useMutation<CreateStripeCheckoutMutation>(CREATE_STRIPE_CHECKOUT);

  const handleCheckout = async (lookUpKey: string) => {
    const stripe = await loadStripe(
      process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || '',
    );
    const response = await createCheckoutSession({
      variables: { teamSlug, lookUpKey },
    });
    const sessionId = response.data?.createCheckoutSession?.sessionId || "";
    const res = await stripe?.redirectToCheckout({ sessionId });
    if (res?.error) {
      console.log('[stripe error]', res.error.message);
    }
  };

  const resourcesUsed = subscriptionData?.resourcesUsed;
  const resourcesIncluded = subscriptionData?.resourcesIncluded;

  return (
    <div className="w-full h-full flex flex-col gap-3">
      {subActive ? (
        <div className="flex flex-row gap-2">
          <Badge variant={'outline'}>{titleCase(subStatus!)}</Badge>
          {subscriptionData?.planName}
        </div>
      ) : (
        <div className="flex flex-row gap-2">
          <Badge variant={'outline'}>Inactive</Badge>
          No Active Subscription
        </div>
      )}

      {subActive && (
        <div className="w-full flex flex-col space-x-2">
          <text fontWeight={'semibold'}>Current Billing Cycle:</text>
          <text fontWeight={'semibold'}>
            {convertUtcToLocal(subscriptionData?.currentPeriodStart || '')} to{' '}
            {convertUtcToLocal(subscriptionData?.currentPeriodEnd || '')}
          </text>
          <div className="mt-2">
            <text fontWeight={'semibold'}>Resources:</text>
            <div className="flex flex-row gap-2">
              <text>{resourcesUsed} used</text>
              <text>of</text>
              <text>{resourcesIncluded} included</text>
            </div>
          </div>
        </div>
      )}

      {subActive ?
        subStatus === 'active' ? (
          <div className="flex flex-col space-y-2">
            <text className="font-semibold">
              Subscription next renews on{' '}
              {convertUtcToLocal(subscriptionData?.currentPeriodEnd || '')}{' '}
              (renews every {subscriptionData?.interval})
            </text>
            <text>
              Card ending with {subscriptionData?.lastFourCardDigits} will be
              charged ${subscriptionData?.costInUsd}
            </text>
            <text>Click on Manage Billing to update payment information</text>
          </div>
        ) : (
          <div className="flex flex-col space-y-2">
            <text className="font-semibold">
              There is a problem in renewing your subscription.
            </text>
            <text className="font-semibold">
              Please click on Manage Billing to resolve the issue.
            </text>
          </div >
        )
        :
        <div />
      }


      <StripeCheckoutForm
        teamSlug={teamSlug}
        handleCheckout={handleCheckout}
        subscriptionActive={subActive}
      />
    </div >
  );
};

function convertUtcToLocal(utcTimestamp: string): string {
  if (utcTimestamp === '') {
    return '';
  }

  // Create a Date object using the provided UTC timestamp
  const date = new Date(utcTimestamp);

  // Get the month, day, and year
  const monthNames = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
  ];
  const month = monthNames[date.getMonth()];
  const day = date.getDate();
  const year = date.getFullYear();

  // Format the date string
  const formattedDate = `${month} ${day} ${year}`;

  return formattedDate;
}
