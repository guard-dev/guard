import PricingTable from '@/components/PricingTable';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import { useMutation, gql } from '@apollo/client';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

const CREATE_STRIPE_PORTAL = gql`
  mutation CreateStripePortalSession($teamSlug: String!) {
    createPortalSession(teamSlug: $teamSlug) {
      sessionUrl
    }
  }
`;

interface StripeCheckoutFormProps {
  teamSlug: string;
  subscriptionActive: boolean;
  handleCheckout: (lookupKey: string) => void;
}

const StripeCheckoutForm: React.FC<StripeCheckoutFormProps> = ({
  teamSlug,
  subscriptionActive,
  handleCheckout,
}) => {
  const router = useRouter();
  const [createPortalSession, { loading: portalLoading, error: portalError }] =
    useMutation(CREATE_STRIPE_PORTAL);

  const handlePortalSession = async () => {
    const response = await createPortalSession({ variables: { teamSlug } });
    const sessionUrl = response.data?.createPortalSession?.sessionUrl;
    if (sessionUrl) {
      router.push(sessionUrl);
    } else {
      console.log('[stripe error]', portalError?.message);
    }
  };

  const [open, setOpen] = useState(false);

  return (
    <div className="w-full flex flex-col md:flex-row overflow-auto gap-2">
      {!subscriptionActive && (
        <Button size="sm" onClick={() => setOpen(true)}>
          Update Subscription
        </Button>
      )}

      <Dialog open={open} onOpenChange={(e) => setOpen(e)}>
        <DialogContent className='max-w-screen-lg'>
          <PricingTable handleCheckout={handleCheckout} />
        </DialogContent>
      </Dialog>

      {subscriptionActive && (
        <Button
          size="sm"
          disabled={portalLoading}
          onClick={handlePortalSession}
        >
          Manage Billing
        </Button>
      )}

      {subscriptionActive && (
        <Button
          variant="destructive"
          size="sm"
          disabled={portalLoading}
          onClick={handlePortalSession}
        >
          Cancel Subscription
        </Button>
      )}
    </div>
  );
};

export default StripeCheckoutForm;
