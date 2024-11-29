# Guard.dev - Open Source Cloud Security Tool

Guard.dev is an open-source, AI-powered cloud security tool designed to scan and secure your cloud environment by identifying misconfigurations and vulnerabilities across AWS. Using advanced large language models (LLMs), Guard.dev provides actionable insights and command-level fixes to enhance cloud security with minimal setup and maintenance.

## Key Features

- **AWS Coverage**: Currently supports AWS services such as IAM, EC2, S3, Lambda, DynamoDB, and ECS, with more services coming soon.
- **AI-Powered Remediation**: Automatically generates suggested command-line fixes and best practices for identified misconfigurations.
- **Real-Time Scanning**: Continuously monitors cloud environments for the latest vulnerabilities and configuration issues.
- **Extensible & Open Source**: Fully open-sourced to allow customization, integration, and community contributions.
- **Flexible Deployment**: Deployable via Docker Compose for a quick and easy setup.

## Installation

### Prebuilt Docker Image

1. **Pull and Run the Docker Image**:

   ```bash
   docker run -d \
       -p 3000:3000 \
       -p 8080:8080 \
       -e GEMINI_SECRET_KEY=your_gemini_secret_key \
       -e AWS_REGION=us-west-2 \
       -e AWS_ACCESS_KEY_ID=your_aws_access_key_id \
       -e AWS_SECRET_ACCESS_KEY=your_aws_secret_access_key \
       ghcr.io/guard-dev/guard:latest
   ```
   Attach the following policy to the IAM User: [templates/guard-self-host-policy.json](./templates/guard-self-host-policy.json).

2. **Access Guard.dev**:
   Open your browser and navigate to `http://localhost:3000`.

### Build From Source

1. **Clone the Repository**:

   ```bash
   git clone https://github.com/guard-dev/guard.git
   cd guard
   ```

2. **Copy Environment Variables File**:

   ```bash
   cp example.env .env
   ```

   - Edit the `.env` file to configure the necessary variables such as `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `GEMINI_SECRET_KEY`, etc.

3. **Run with Docker Compose**:

   ```bash
   ./start # Ensure that Docker is running and Python3 is installed.
   ```

4. **Access Guard.dev**:
   Open your browser and navigate to `http://localhost:3000`.

## Environment Variable Setup

To set up Guard.dev for use, you will need to configure the following services:

### 1. AWS Account Linking

For self-hosting Guard.dev, follow these steps to set up an IAM User with the required permissions:

1. **Create an IAM User** in your AWS account with **programmatic access**.
2. **Attach the Policy** to the IAM User:
   - Download or copy the policy document from [templates/guard-self-host-policy.json](./templates/guard-self-host-policy.json).
   - Attach this policy to the IAM User during or after creation.
3. **Obtain Access Keys**:
   - After creating the IAM User, make sure to securely store the **Access Key ID** and **Secret Access Key**.
4. **Configure Environment Variables**:
   - Set the following environment variables in your hosting environment:
     - `AWS_ACCESS_KEY_ID`
     - `AWS_SECRET_ACCESS_KEY`

This IAM User will have the necessary permissions for Guard to perform cloud security scans across your AWS resources.

### 2. LLM Scanning with Google Gemini

Guard.dev leverages Google Gemini for LLM-based scans.

- Follow the guide at [Google Gemini API Key Setup](https://ai.google.dev/gemini-api/docs/api-key) to obtain an API key.
- Set the API key as an environment variable named `GEMINI_SECRET_KEY`.
- Make sure the API key is properly secured and included in the `.env` file.

### Environment Variables

Update the `.env` file with the following environment variables:

- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` for AWS Integration.
- `GEMINI_SECRET_KEY` for LLM scans.

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
- **Support**: <support@guard.dev>
