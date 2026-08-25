import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router";

import { useApiClient } from "../api/client";
import { applyContractErrors } from "../api/formErrors";
import { unwrap } from "../api/generated/client";

type ProductFormValues = {
  name: string;
  price: number;
};

export function ProductFormPage() {
  const client = useApiClient();
  const navigate = useNavigate();
  const [status, setStatus] = useState("");
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProductFormValues>({
    defaultValues: { name: "", price: 0 },
  });

  async function onSubmit(values: ProductFormValues) {
    setStatus("");
    try {
      await unwrap(
        await client.POST("/api/v1/products", {
          body: { name: values.name, price: values.price },
        }),
      );
      navigate("/");
    } catch (err: unknown) {
      if (!applyContractErrors(setError, err)) {
        setStatus(err instanceof Error ? err.message : "request failed");
      }
    }
  }

  return (
    <section>
      <h1>New product</h1>
      <p>
        <Link to="/">Back to list</Link>
      </p>
      <form onSubmit={handleSubmit(onSubmit)}>
        <label>
          Name
          <input type="text" {...register("name", { required: "Name is required" })} />
        </label>
        {errors.name?.message ? <p>{errors.name.message}</p> : null}
        <label>
          Price
          <input type="number" {...register("price", { setValueAs: (value) => (value === "" ? 0 : Number(value)) })} />
        </label>
        {errors.price?.message ? <p>{errors.price.message}</p> : null}
        <button type="submit" disabled={isSubmitting}>
          Create
        </button>
      </form>
      {status ? <p>{status}</p> : null}
    </section>
  );
}
