# 📌 Feature Spec: Input Burst

## 🔄 Burst Support

**Burst simulation** is currently implemented only at the **generator stage**, which feeds items into the pipeline.

### Key Details:

* **`InputBurst`**: User-defined function that returns a batch of items.
* **`BurstInterval`** & `BurstCount`: Control how often and how many bursts occur.
* **Purpose**: Simulate traffic spikes and evaluate system resilience under sudden load.
