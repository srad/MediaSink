use serde::{Deserialize, Deserializer};

pub fn null_default<'de, D, T>(deserializer: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Default + Deserialize<'de>,
{
    Option::<T>::deserialize(deserializer).map(|value| value.unwrap_or_default())
}

#[cfg(test)]
mod tests {
    use super::null_default;
    use serde::Deserialize;

    #[derive(Debug, Deserialize)]
    struct Wrapper {
        #[serde(default, deserialize_with = "null_default")]
        values: Vec<u64>,
    }

    #[test]
    fn treats_null_as_default() {
        let parsed: Wrapper = serde_json::from_str(r#"{"values":null}"#).expect("valid json");
        assert!(parsed.values.is_empty());
    }
}
