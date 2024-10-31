# Guard.dev - Open Source Cloud Security Tool

![Guard.dev Logo](link-to-logo)

Guard.dev is an open-source, AI-powered cloud security tool designed to scan and secure your cloud environment by identifying misconfigurations and vulnerabilities across AWS. Using advanced large language models (LLMs), Guard.dev provides actionable insights and command-level fixes to enhance cloud security with minimal setup and maintenance.

## Key Features

- **AWS Coverage**: Currently supports AWS services such as IAM, EC2, S3, Lambda, DynamoDB, and ECS, with more services coming soon.
- **AI-Powered Remediation**: Automatically generates suggested command-line fixes and best practices for identified misconfigurations.
- **Real-Time Scanning**: Continuously monitors cloud environments for the latest vulnerabilities and configuration issues.
- **Extensible & Open Source**: Fully open-sourced to allow customization, integration, and community contributions.
- **Flexible Deployment**: Deployable via Docker Compose for a quick and easy setup.

## Pricing & Plans

Guard.dev is free to use in its open-source form, with additional paid plans that offer extended support and scalability for large cloud environments.

| Plan       | Price      | Resources       | Additional Resources | Support          |
|------------|------------|-----------------|----------------------|-------------------|
| **Open Source** | Free       | Unlimited       | -                    | Community-Supported |
| **Basic**       | $150/month | 1,000 resources | $6 per 100 extra     | Email Support     |
| **Pro**         | $400/month | 5,000 resources | $5 per 100 extra     | Email & Chat Support |
| **Enterprise**  | Contact Us | Custom          | Custom               | Dedicated Support |

## Installation

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/yourusername/guard.dev.git
   cd guard.dev
   ```

2. **Run with Docker Compose**:
   ```bash
   docker-compose up -d
   ```

3. **Access Guard.dev**:
   Open your browser and navigate to `http://localhost:PORT` (Replace `PORT` with the configured port in `docker-compose.yml`).

4. **Configure Cloud Accounts**:
   - Integrate your cloud accounts by following the setup guide in the `docs/setup.md`.

## Usage

1. **Authenticate** with your AWS cloud account.
2. **Run Scans** on specific services or across the entire environment.
3. **Review Findings** and generated fixes for misconfigurations and vulnerabilities.
4. **Implement Fixes** using the provided commands or export a summary report.

## Supported Services

### AWS
- IAM, EC2, S3, Lambda, DynamoDB, ECS (more services coming soon)

### GCP & Azure
- Support for GCP and Azure is coming soon

## Contributing

We welcome contributions from the community! Please refer to `CONTRIBUTING.md` for guidelines on how to get involved.

## License

Guard.dev is released under the [Server Side Public License (SSPL)](LICENSE). This license allows you to view, modify, and self-host the software, but restricts using it to offer commercial services without open-sourcing your modifications.

## Get in Touch

- **Website**: [www.guard.dev](https://www.guard.dev)
- **Support**: support@guard.dev
- **Twitter**: [@GuardDevSec](https://twitter.com/GuardDevSec)
- **LinkedIn**: [Guard.dev](https://linkedin.com/company/guarddev)
