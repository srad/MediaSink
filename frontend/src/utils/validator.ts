export interface FieldConfig {
  name: string;

  validator(t: unknown): boolean;

  validMessage: string;
  invalidMessage: string;
}

export interface ValidationMessage {
  message: string;
  isValid: boolean;
}

class FieldValidator<T> {
  private readonly fieldConfig: { [key: string]: FieldConfig } = {};

  constructor(config: FieldConfig[]) {
    config.forEach((fieldConfig) => (this.fieldConfig[fieldConfig.name] = fieldConfig));
  }

  validate(name: string, value: T): ValidationMessage {
    if (!this.fieldConfig[name]) {
      throw new Error(`Invalid field config: ${this.fieldConfig[name]}`);
    }

    const isValidTest = this.fieldConfig[name].validator(value);

    const message = isValidTest ? this.fieldConfig[name]!.validMessage : this.fieldConfig[name]!.invalidMessage;
    return { isValid: isValidTest, message: `${message}: "${value}"` };
  }

  validateAll(validationRequest: { name: string; value: T }[]): ValidationMessage[] {
    return validationRequest.map((request) => this.validate(request.name, request.value));
  }
}

export const createValidator = function (config: FieldConfig[]) {
  return new FieldValidator(config);
};
