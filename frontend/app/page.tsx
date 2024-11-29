import { Button } from "@/components/ui/button";
import PricingTable from "@/components/PricingTable";
import { IconCloudSecurity, IconAI, IconRealTime, IconActionable } from '@/components/Icons';
import Navbar from "./console/Navbar";
import { FooterSection } from "./console/Footer";

export default function Home() {
  return (
    <div className="flex font-[family-name:var(--font-geist-sans)] bg-background min-h-screen flex-col">
      <Navbar />

      {/* Hero Section */}
      <section className="w-full bg-gradient-to-b from-background via-background to-neutral-100/50 dark:to-neutral-900/50 pt-24 md:pt-32 pb-32">
        <div className="flex-1 flex flex-col items-center w-full max-w-screen-2xl mx-auto px-6">
          <HeroSection />
        </div>
      </section>

      {/* Features Section */}
      <section className="w-full bg-gradient-to-b from-neutral-100/50 via-neutral-100/50 to-background dark:from-neutral-900/50 dark:via-neutral-900/50 dark:to-background py-20">
        <div className="flex-1 flex flex-col items-center w-full max-w-screen-2xl mx-auto px-6">
          <div className="w-full max-w-[1024px]">
            <FeaturesSection />
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section className="w-full bg-gradient-to-b from-background via-background to-transparent dark:from-background dark:via-background dark:to-transparent pt-20 pb-24">
        <div className="flex-1 flex flex-col items-center w-full max-w-screen-2xl mx-auto px-6">
          <div className="w-full max-w-[1024px]">
            <PricingTable />
          </div>
        </div>
      </section>

      <div className="mt-auto">
        <FooterSection />
      </div>
    </div>
  );
}

const HeroSection = () => {
  return (
    <div className="flex flex-col items-center text-center gap-8 max-w-[800px]">
      <div className="space-y-4">
        <h1 className="text-6xl md:text-7xl font-[family-name:var(--font-geist-mono)] text-foreground">
          guard
        </h1>
        <h2 className="text-2xl md:text-3xl font-[family-name:var(--font-geist-mono)] text-foreground/80">
          AI-Powered Cloud Security, Simplified.
        </h2>
        <p className="text-lg font-[family-name:var(--font-geist-mono)] text-foreground/60 max-w-[600px] mx-auto">
          Detect AWS misconfigurations in real-time and fix them with AI-driven, actionable insights.
        </p>
      </div>

      <div className="flex flex-wrap gap-4 justify-center">
        <a href="/console">
          <Button size="lg" className="rounded-full bg-gradient-to-r from-neutral-900 via-neutral-800 to-neutral-700 hover:from-neutral-800 hover:via-neutral-700 hover:to-neutral-600 text-white border-0 dark:from-white dark:via-neutral-200 dark:to-neutral-400 dark:hover:from-neutral-200 dark:hover:via-neutral-300 dark:hover:to-neutral-500 dark:text-neutral-900">
            Get Started
          </Button>
        </a>
        <a href="https://www.github.com/guard-dev/guard" target="_blank" rel="noopener noreferrer">
          <Button size="lg" className="rounded-full bg-gradient-to-r from-neutral-100 via-white to-neutral-200 hover:from-white hover:via-neutral-100 hover:to-neutral-300 dark:from-neutral-800 dark:via-neutral-900 dark:to-neutral-950 dark:hover:from-neutral-700 dark:hover:via-neutral-800 dark:hover:to-neutral-900 border border-neutral-200 dark:border-neutral-700 text-neutral-900 dark:text-white">
            Self Host
          </Button>
        </a>
        <a href="https://discord.gg/XCKnF8k2hF" target="_blank" rel="noopener noreferrer">
          <Button size="lg" className="rounded-full bg-gradient-to-r from-neutral-100 via-white to-neutral-200 hover:from-white hover:via-neutral-100 hover:to-neutral-300 dark:from-neutral-800 dark:via-neutral-900 dark:to-neutral-950 dark:hover:from-neutral-700 dark:hover:via-neutral-800 dark:hover:to-neutral-900 border border-neutral-200 dark:border-neutral-700 text-neutral-900 dark:text-white">
            Community Discord
          </Button>
        </a>
      </div>
    </div>
  );
}

const FeaturesSection = () => {
  const features = [
    {
      icon: <IconCloudSecurity className="w-12 h-12 text-neutral-800 dark:text-neutral-200" />,
      title: "Cloud Security Scanning",
      description: "Comprehensive scanning of your AWS resources to identify potential security risks and misconfigurations."
    },
    {
      icon: <IconAI className="w-12 h-12 text-neutral-700 dark:text-neutral-300" />,
      title: "AI-Powered Analysis",
      description: "Leverage advanced LLMs to process and analyze your cloud infrastructure for deeper insights."
    },
    {
      icon: <IconRealTime className="w-12 h-12 text-neutral-800 dark:text-neutral-200" />,
      title: "Real-Time Detection",
      description: "Instantly identify vulnerabilities and security issues as they arise in your AWS environment."
    },
    {
      icon: <IconActionable className="w-12 h-12 text-neutral-700 dark:text-neutral-300" />,
      title: "Actionable Insights",
      description: "Receive clear, actionable recommendations to improve your cloud security posture."
    }
  ];

  return (
    <div className="w-full">
      <h2 className="text-3xl font-[family-name:var(--font-geist-mono)] mb-12 text-center">Features</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {features.map((feature, index) => (
          <div key={index} className="p-6 rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 hover:shadow-lg transition-all duration-300">
            {feature.icon}
            <h3 className="text-xl font-[family-name:var(--font-geist-mono)] mb-2">{feature.title}</h3>
            <p className="text-foreground/60">{feature.description}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
